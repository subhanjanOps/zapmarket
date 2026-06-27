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
)

// Per-call timeouts for downstream gRPC calls in the checkout saga.
// These prevent a hung downstream service from blocking the order request indefinitely.
const (
	catalogCallTimeout   = 5 * time.Second
	inventoryCallTimeout = 5 * time.Second
	paymentCallTimeout   = 30 * time.Second
)

// inventoryGateway is the subset of clients.InventoryClient the saga needs.
// Keeping it here avoids importing the clients package in tests.
type inventoryGateway interface {
	ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (uuid.UUID, bool, error)
	ReleaseStock(ctx context.Context, reservationID uuid.UUID) error
	DeductStock(ctx context.Context, reservationID uuid.UUID) error
}

// paymentGateway is the subset of clients.PaymentClient the saga needs.
type paymentGateway interface {
	ChargeCard(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (uuid.UUID, string, error)
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
	payment   paymentGateway
	catalog   catalogGateway
	cache     contracts.OrderCache
	logger    *slog.Logger
}

func NewOrderService(
	repo contracts.OrderRepository,
	inventory inventoryGateway,
	payment paymentGateway,
	catalog catalogGateway,
	cache contracts.OrderCache,
	logger *slog.Logger,
) OrderService {
	return &orderService{repo: repo, inventory: inventory, payment: payment, catalog: catalog, cache: cache, logger: logger}
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

	order := &domain.Order{UserID: userID, IdempotencyKey: idempotencyKey, Status: domain.OrderPending, TotalAmount: totalAmount, Currency: currency}
	if err := s.repo.CreateOrder(ctx, order, domainItems); err != nil {
		return nil, err
	}
	s.logger.Info("order created", "order_id", order.ID, "total", totalAmount)

	if err := s.reserveStockForOrder(ctx, order.ID, domainItems); err != nil {
		return nil, err
	}
	order.Status = domain.OrderReserved
	s.logger.Info("order reserved", "order_id", order.ID)

	paymentCtx, paymentCancel := context.WithTimeout(ctx, paymentCallTimeout)
	paymentID, err := s.finalisePayment(paymentCtx, order, domainItems, userID, currency, idempotencyKey, paymentMethodID)
	paymentCancel()
	if err != nil {
		return nil, err
	}

	order.Status = domain.OrderConfirmed
	order.PaymentID = &paymentID
	s.logger.Info("order confirmed", "order_id", order.ID, "payment_id", paymentID)

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

// reserveStockForOrder reserves inventory for all items, using per-call timeouts.
// Compensates and returns error on failure.
func (s *orderService) reserveStockForOrder(ctx context.Context, orderID uuid.UUID, items []*domain.OrderItem) error {
	var reserved []*domain.OrderItem
	for _, item := range items {
		invCtx, invCancel := context.WithTimeout(ctx, inventoryCallTimeout)
		reservationID, ok, err := s.inventory.ReserveStock(invCtx, item.SKUID, orderID, item.Quantity)
		invCancel()
		if err != nil {
			s.logger.Error("inventory ReserveStock error", "sku_id", item.SKUID, "error", err)
			s.compensate(ctx, orderID, reserved)
			return pkgerrors.NewInternal("INVENTORY_ERROR", "failed to reserve stock", err)
		}
		if !ok {
			s.logger.Info("insufficient stock", "sku_id", item.SKUID)
			s.compensate(ctx, orderID, reserved)
			return pkgerrors.NewConflict("INSUFFICIENT_STOCK", "insufficient stock for sku "+item.SKUID.String())
		}
		item.ReservationID = &reservationID
		reserved = append(reserved, item)
	}
	if err := s.repo.MarkReserved(ctx, orderID, items); err != nil {
		s.compensate(ctx, orderID, reserved)
		return err
	}
	return nil
}

// finalisePayment charges the card and, on success, deducts stock and confirms the order.
// On failure it compensates inventory and marks the order cancelled before returning.
func (s *orderService) finalisePayment(ctx context.Context, order *domain.Order, items []*domain.OrderItem, userID uuid.UUID, currency string, idempotencyKey uuid.UUID, paymentMethodID string) (uuid.UUID, error) {
	paymentID, paymentStatus, err := s.payment.ChargeCard(ctx, order.ID, userID, order.TotalAmount, currency, idempotencyKey, paymentMethodID)
	if err != nil {
		s.logger.Error("payment ChargeCard error", "order_id", order.ID, "error", err)
		s.compensate(ctx, order.ID, items)
		s.cancelWithPayload(ctx, order.ID, map[string]string{"order_id": order.ID.String(), "user_id": userID.String(), "reason": "payment_error"})
		return uuid.Nil, pkgerrors.NewInternal("PAYMENT_ERROR", "payment service error", err)
	}
	if paymentStatus != "CAPTURED" {
		s.logger.Info("payment not captured", "order_id", order.ID, "payment_status", paymentStatus)
		s.compensate(ctx, order.ID, items)
		s.cancelWithPayload(ctx, order.ID, map[string]string{"order_id": order.ID.String(), "user_id": userID.String(), "reason": "payment_failed", "payment_status": paymentStatus})
		return uuid.Nil, pkgerrors.NewConflict("PAYMENT_FAILED", "payment was not captured (status: "+paymentStatus+")")
	}

	for _, item := range items {
		if item.ReservationID == nil {
			continue
		}
		invCtx, invCancel := context.WithTimeout(ctx, inventoryCallTimeout)
		if err := s.inventory.DeductStock(invCtx, *item.ReservationID); err != nil {
			s.logger.Error("failed to deduct stock after payment", "reservation_id", item.ReservationID, "error", err)
		}
		invCancel()
	}

	confirmPayload, marshalErr := json.Marshal(map[string]string{"order_id": order.ID.String(), "user_id": userID.String(), "payment_id": paymentID.String()})
	if marshalErr != nil {
		return uuid.Nil, pkgerrors.NewInternal("INTERNAL_ERROR", "internal serialization error", marshalErr)
	}
	if err := s.repo.MarkConfirmed(ctx, order.ID, paymentID, confirmPayload); err != nil {
		return uuid.Nil, err
	}
	return paymentID, nil
}

// cancelWithPayload marshals the payload and marks the order cancelled, logging any error.
func (s *orderService) cancelWithPayload(ctx context.Context, orderID uuid.UUID, payload map[string]string) {
	b, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		s.logger.Error("failed to marshal cancel payload", "order_id", orderID, "error", marshalErr)
		return
	}
	if err := s.repo.MarkCancelled(ctx, orderID, b); err != nil {
		s.logger.Error("failed to mark order cancelled", "order_id", orderID, "error", err)
	}
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
