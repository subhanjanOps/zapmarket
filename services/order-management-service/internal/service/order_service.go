package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain/contracts"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain/events"
)

// Per-call timeout for catalog gRPC calls in the checkout path.
// Prevents a hung catalog service from blocking the order request indefinitely.
const catalogCallTimeout = 5 * time.Second

// inventoryGateway is the subset of clients.InventoryClient the saga needs.
// Keeping it here avoids importing the clients package in tests.
type inventoryGateway interface {
	ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (uuid.UUID, bool, error)
	ReleaseStock(ctx context.Context, reservationID uuid.UUID) error
	DeductStock(ctx context.Context, reservationID uuid.UUID) error
}

// catalogGateway fetches authoritative SKU prices to prevent client-supplied price injection.
type catalogGateway interface {
	GetSKUPrice(ctx context.Context, skuID uuid.UUID) (int64, error)
}

// OrderService defines the public interface for order operations.
type OrderService interface {
	Checkout(ctx context.Context, userID, idempotencyKey uuid.UUID, items []CheckoutItem, currency, paymentMethodID string) (*domain.Order, error)
	GetOrder(ctx context.Context, orderID, userID uuid.UUID) (*domain.Order, []*domain.OrderItem, error)
	ListOrders(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Order, int64, error)
	CancelOrder(ctx context.Context, orderID, userID uuid.UUID) (*domain.Order, error)

	// ListSellerOrders returns paginated orders containing at least one item from the seller.
	ListSellerOrders(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]*domain.Order, int64, error)
	// GetSellerOrder returns a single order + its items if it contains the seller's SKUs.
	GetSellerOrder(ctx context.Context, orderID, sellerID uuid.UUID) (*domain.Order, []*domain.OrderItem, error)

	// Admin operations — no ownership checks.
	ListAllOrders(ctx context.Context, params contracts.OrderListParams) ([]*domain.Order, int64, error)
	AdminGetOrder(ctx context.Context, orderID uuid.UUID) (*domain.Order, []*domain.OrderItem, error)
	AdminCancelOrder(ctx context.Context, orderID uuid.UUID) (*domain.Order, error)
}

// CheckoutItem is the per-SKU input to Checkout.
type CheckoutItem struct {
	SKUID     uuid.UUID
	SellerID  *uuid.UUID
	Quantity  int
	UnitPrice int64
}

const idempotencyTTL = 24 * time.Hour

type orderService struct {
	repo      contracts.OrderRepository
	inventory inventoryGateway
	catalog   catalogGateway
	cache     contracts.OrderCache
	logger    *slog.Logger
}

func NewOrderService(
	repo contracts.OrderRepository,
	inventory inventoryGateway,
	catalog catalogGateway,
	cache contracts.OrderCache,
	logger *slog.Logger,
) OrderService {
	return &orderService{repo: repo, inventory: inventory, catalog: catalog, cache: cache, logger: logger}
}

func idempCacheKey(key uuid.UUID) string {
	return fmt.Sprintf("idempotency:order:%s", key)
}

func (s *orderService) Checkout(ctx context.Context, userID, idempotencyKey uuid.UUID, items []CheckoutItem, currency, paymentMethodID string) (*domain.Order, error) {
	if userID == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "user_id is required")
	}
	if idempotencyKey == uuid.Nil {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "idempotency_key is required")
	}
	if len(items) == 0 {
		return nil, pkgerrors.NewValidation("INVALID_DATA", "order must have at least one item")
	}
	if currency == "" {
		currency = "INR"
	}

	if existing, err := s.checkIdempotency(ctx, idempotencyKey); existing != nil || err != nil {
		return existing, err
	}

	// Build domain items + total. Prices are fetched from the authoritative
	// catalog to prevent client-supplied price injection.
	var totalAmount int64
	domainItems := make([]*domain.OrderItem, len(items))
	for i, it := range items {
		if it.Quantity <= 0 {
			return nil, pkgerrors.NewValidation("INVALID_DATA", "item quantity must be greater than zero")
		}
		catalogCtx, catalogCancel := context.WithTimeout(ctx, catalogCallTimeout)
		authPrice, err := s.catalog.GetSKUPrice(catalogCtx, it.SKUID)
		catalogCancel()
		if err != nil {
			return nil, pkgerrors.NewValidation("INVALID_SKU", fmt.Sprintf("SKU %s not found or unavailable: %v", it.SKUID, err))
		}
		if authPrice <= 0 {
			return nil, pkgerrors.NewValidation("INVALID_SKU", fmt.Sprintf("SKU %s has no valid price configured", it.SKUID))
		}
		totalAmount += int64(it.Quantity) * authPrice
		domainItems[i] = &domain.OrderItem{SKUID: it.SKUID, SellerID: it.SellerID, Quantity: it.Quantity, UnitPrice: authPrice}
	}

	order := &domain.Order{
		ID:             uuid.New(), // pre-assign so the event payload carries the correct order_id
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
		Status:         domain.OrderPending,
		SagaStatus:     "AWAITING_INVENTORY",
		TotalAmount:    totalAmount,
		Currency:       currency,
	}

	// Build the checkout.requested event payload before the DB write so the
	// outbox row is written atomically with the order — no separate Kafka publish.
	checkoutItems := make([]events.CheckoutItem, len(domainItems))
	for i, it := range domainItems {
		checkoutItems[i] = events.CheckoutItem{
			SkuID:    it.SKUID.String(),
			Quantity: it.Quantity,
		}
	}
	evt := events.CheckoutRequestedEvent{
		OrderID:         order.ID.String(),
		UserID:          userID.String(),
		Items:           checkoutItems,
		RequestedAt:     time.Now(),
		AmountCents:     totalAmount,
		Currency:        currency,
		PaymentMethodID: paymentMethodID,
	}
	outboxPayload, err := json.Marshal(evt)
	if err != nil {
		return nil, pkgerrors.NewInternal("INTERNAL_ERROR", "failed to marshal checkout event", err)
	}

	if err := s.repo.CreateOrder(ctx, order, domainItems, outboxPayload); err != nil {
		return nil, err
	}
	s.logger.Info("order created — checkout.requested written to outbox", "order_id", order.ID, "total", totalAmount)

	// Cache the PENDING order so duplicate requests with the same idempotency
	// key get an immediate response without hitting the DB unique constraint.
	if b, marshalErr := json.Marshal(order); marshalErr == nil {
		_ = s.cache.Set(ctx, idempCacheKey(idempotencyKey), b, idempotencyTTL)
	}

	return order, nil
}

// checkIdempotency returns an existing order if the idempotency key was already used, or (nil, nil) to proceed.
func (s *orderService) checkIdempotency(ctx context.Context, idempotencyKey uuid.UUID) (*domain.Order, error) {
	cacheKey := idempCacheKey(idempotencyKey)
	lockKey := cacheKey + ":lock"
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var order domain.Order
		if json.Unmarshal(cached, &order) == nil {
			s.logger.Info("idempotent replay from cache", "order_id", order.ID)
			return &order, nil
		}
	}

	const lockTTL = 10 * time.Second
	acquired, err := s.cache.SetNX(ctx, lockKey, "1", lockTTL)
	if err != nil {
		s.logger.Warn("idempotency lock unavailable, proceeding without lock", "error", err)
	}
	defer func() {
		if acquired {
			_ = s.cache.Del(ctx, lockKey)
		}
	}()

	existing, err := s.repo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err == nil {
		s.logger.Info("idempotent replay from db", "order_id", existing.ID, "status", existing.Status)
		if b, marshalErr := json.Marshal(existing); marshalErr == nil {
			_ = s.cache.Set(ctx, cacheKey, b, idempotencyTTL)
		}
		return existing, nil
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != pkgerrors.NotFound {
		return nil, err
	}
	return nil, nil
}

func (s *orderService) GetOrder(ctx context.Context, orderID, userID uuid.UUID) (*domain.Order, []*domain.OrderItem, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}
	if order.UserID != userID {
		return nil, nil, pkgerrors.NewNotFound("ORDER_NOT_FOUND", "order not found")
	}
	items, err := s.repo.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}
	return order, items, nil
}

func (s *orderService) ListOrders(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Order, int64, error) {
	return s.repo.GetByUserID(ctx, userID, contracts.OrderPageParams{Limit: limit, Offset: offset})
}

func (s *orderService) CancelOrder(ctx context.Context, orderID, userID uuid.UUID) (*domain.Order, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, pkgerrors.NewNotFound("ORDER_NOT_FOUND", "order not found")
	}
	if err := order.Transition(domain.OrderCancelled); err != nil {
		return nil, err
	}

	items, err := s.repo.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, err
	}

	cancelPayload, marshalErr := json.Marshal(map[string]string{"order_id": orderID.String(), "user_id": userID.String(), "reason": "user_requested"})
	if marshalErr != nil {
		return nil, pkgerrors.NewInternal("INTERNAL_ERROR", "internal serialization error", marshalErr)
	}
	if err := s.repo.MarkCancelled(ctx, orderID, cancelPayload); err != nil {
		return nil, err
	}

	s.compensate(ctx, orderID, items)

	order.Status = domain.OrderCancelled
	s.logger.Info("order cancelled by user", "order_id", orderID)
	return order, nil
}

func (s *orderService) ListSellerOrders(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]*domain.Order, int64, error) {
	return s.repo.GetBySellerID(ctx, sellerID, contracts.OrderPageParams{Limit: limit, Offset: offset})
}

func (s *orderService) GetSellerOrder(ctx context.Context, orderID, sellerID uuid.UUID) (*domain.Order, []*domain.OrderItem, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.repo.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}
	hasSeller := false
	for _, item := range items {
		if item.SellerID != nil && *item.SellerID == sellerID {
			hasSeller = true
			break
		}
	}
	if !hasSeller {
		return nil, nil, pkgerrors.NewNotFound("ORDER_NOT_FOUND", "order not found")
	}
	return order, items, nil
}

func (s *orderService) ListAllOrders(ctx context.Context, params contracts.OrderListParams) ([]*domain.Order, int64, error) {
	return s.repo.ListAll(ctx, params)
}

func (s *orderService) AdminGetOrder(ctx context.Context, orderID uuid.UUID) (*domain.Order, []*domain.OrderItem, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.repo.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, nil, err
	}
	return order, items, nil
}

func (s *orderService) AdminCancelOrder(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if err := order.Transition(domain.OrderCancelled); err != nil {
		return nil, err
	}

	items, err := s.repo.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, err
	}

	cancelPayload, marshalErr := json.Marshal(map[string]string{"order_id": orderID.String(), "reason": "admin_cancelled"})
	if marshalErr != nil {
		return nil, pkgerrors.NewInternal("INTERNAL_ERROR", "internal serialization error", marshalErr)
	}
	if err := s.repo.MarkCancelled(ctx, orderID, cancelPayload); err != nil {
		return nil, err
	}

	s.compensate(ctx, orderID, items)

	order.Status = domain.OrderCancelled
	s.logger.Info("order admin-cancelled", "order_id", orderID)
	return order, nil
}

// compensate releases all reservations that were already made before a
// failure. Errors are logged but not returned — the caller's error takes
// precedence. A failed release means stock is temporarily stranded in
// RESERVED state; it will be recovered by the TTL sweep job.
func (s *orderService) compensate(ctx context.Context, orderID uuid.UUID, reserved []*domain.OrderItem) {
	compensateCtx := context.WithoutCancel(ctx)
	for _, item := range reserved {
		if item.ReservationID == nil {
			continue
		}
		if err := s.inventory.ReleaseStock(compensateCtx, *item.ReservationID); err != nil {
			s.logger.Error("compensation: failed to release reservation",
				"order_id", orderID,
				"reservation_id", item.ReservationID,
				"error", err,
			)
		}
	}
}
