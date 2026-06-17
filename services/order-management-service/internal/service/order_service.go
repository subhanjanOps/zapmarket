package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/clients"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain/contracts"
)

// OrderService defines the public interface for order operations.
type OrderService interface {
	Checkout(ctx context.Context, userID, idempotencyKey uuid.UUID, items []CheckoutItem, currency string) (*domain.Order, error)
	GetOrder(ctx context.Context, orderID, userID uuid.UUID) (*domain.Order, []*domain.OrderItem, error)
	ListOrders(ctx context.Context, userID uuid.UUID) ([]*domain.Order, error)
	CancelOrder(ctx context.Context, orderID, userID uuid.UUID) (*domain.Order, error)
}

// CheckoutItem is the per-SKU input to Checkout.
type CheckoutItem struct {
	SKUID     uuid.UUID
	Quantity  int
	UnitPrice int64
}

const idempotencyTTL = 24 * time.Hour

type orderService struct {
	repo      contracts.OrderRepository
	inventory *clients.InventoryClient
	payment   *clients.PaymentClient
	rdb       *redis.Client
	logger    *slog.Logger
}

func NewOrderService(
	repo contracts.OrderRepository,
	inventory *clients.InventoryClient,
	payment *clients.PaymentClient,
	rdb *redis.Client,
	logger *slog.Logger,
) OrderService {
	return &orderService{repo: repo, inventory: inventory, payment: payment, rdb: rdb, logger: logger}
}

func idempCacheKey(key uuid.UUID) string {
	return fmt.Sprintf("idempotency:order:%s", key)
}

func (s *orderService) Checkout(ctx context.Context, userID, idempotencyKey uuid.UUID, items []CheckoutItem, currency string) (*domain.Order, error) {
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

	// Idempotency: Redis-first cache check (24h TTL), DB fallback.
	cacheKey := idempCacheKey(idempotencyKey)
	if cached, err := s.rdb.Get(ctx, cacheKey).Bytes(); err == nil {
		var order domain.Order
		if json.Unmarshal(cached, &order) == nil {
			s.logger.Info("idempotent replay from cache", "order_id", order.ID)
			return &order, nil
		}
	}
	existing, err := s.repo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err == nil {
		s.logger.Info("idempotent replay from db", "order_id", existing.ID, "status", existing.Status)
		if b, err := json.Marshal(existing); err == nil {
			_ = s.rdb.Set(ctx, cacheKey, b, idempotencyTTL).Err()
		}
		return existing, nil
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != pkgerrors.NotFound {
		return nil, err
	}

	// Step 1: persist order in PENDING + items.
	var totalAmount int64
	domainItems := make([]*domain.OrderItem, len(items))
	for i, it := range items {
		if it.Quantity <= 0 {
			return nil, pkgerrors.NewValidation("INVALID_DATA", "item quantity must be greater than zero")
		}
		if it.UnitPrice <= 0 {
			return nil, pkgerrors.NewValidation("INVALID_DATA", "item unit_price must be greater than zero")
		}
		totalAmount += int64(it.Quantity) * it.UnitPrice
		domainItems[i] = &domain.OrderItem{
			SKUID:     it.SKUID,
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice,
		}
	}

	order := &domain.Order{
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
		Status:         domain.OrderPending,
		TotalAmount:    totalAmount,
		Currency:       currency,
	}
	if err := s.repo.CreateOrder(ctx, order, domainItems); err != nil {
		return nil, err
	}
	s.logger.Info("order created", "order_id", order.ID, "total", totalAmount)

	// Step 2: reserve stock for each item via inventory gRPC.
	var reserved []*domain.OrderItem
	for _, item := range domainItems {
		reservationID, ok, err := s.inventory.ReserveStock(ctx, item.SKUID, order.ID, item.Quantity)
		if err != nil {
			s.logger.Error("inventory ReserveStock error", "sku_id", item.SKUID, "error", err)
			s.compensate(ctx, order.ID, reserved)
			return nil, pkgerrors.NewInternal("INVENTORY_ERROR", "failed to reserve stock", err)
		}
		if !ok {
			s.logger.Info("insufficient stock", "sku_id", item.SKUID)
			s.compensate(ctx, order.ID, reserved)
			return nil, pkgerrors.NewConflict("INSUFFICIENT_STOCK", "insufficient stock for sku "+item.SKUID.String())
		}
		item.ReservationID = &reservationID
		reserved = append(reserved, item)
	}

	// Step 3: persist RESERVED status + reservation IDs.
	if err := s.repo.MarkReserved(ctx, order.ID, domainItems); err != nil {
		s.compensate(ctx, order.ID, reserved)
		return nil, err
	}
	order.Status = domain.OrderReserved
	s.logger.Info("order reserved", "order_id", order.ID)

	// Step 4: charge payment — outside DB transaction.
	paymentID, paymentStatus, err := s.payment.ChargeCard(ctx, order.ID, userID, totalAmount, currency, idempotencyKey)
	if err != nil {
		s.logger.Error("payment ChargeCard error", "order_id", order.ID, "error", err)
		s.compensate(ctx, order.ID, domainItems)
		cancelPayload, _ := json.Marshal(map[string]string{"order_id": order.ID.String(), "user_id": userID.String(), "reason": "payment_error"})
		_ = s.repo.MarkCancelled(ctx, order.ID, cancelPayload)
		return nil, pkgerrors.NewInternal("PAYMENT_ERROR", "payment service error", err)
	}

	// Step 5: handle payment outcome.
	if paymentStatus != "CAPTURED" {
		s.logger.Info("payment not captured", "order_id", order.ID, "payment_status", paymentStatus)
		s.compensate(ctx, order.ID, domainItems)
		cancelPayload, _ := json.Marshal(map[string]string{"order_id": order.ID.String(), "user_id": userID.String(), "reason": "payment_failed", "payment_status": paymentStatus})
		_ = s.repo.MarkCancelled(ctx, order.ID, cancelPayload)
		return nil, pkgerrors.NewConflict("PAYMENT_FAILED", "payment was not captured (status: "+paymentStatus+")")
	}

	// Step 5a: deduct stock from qty_on_hand (finalise the sale).
	for _, item := range domainItems {
		if item.ReservationID == nil {
			continue
		}
		if err := s.inventory.DeductStock(ctx, *item.ReservationID); err != nil {
			// Log and continue — payment already captured; stock discrepancy
			// is reconcilable; do not fail the order over a deduct error.
			s.logger.Error("failed to deduct stock after payment", "reservation_id", item.ReservationID, "error", err)
		}
	}

	// Step 5b: persist CONFIRMED + outbox event.
	confirmPayload, _ := json.Marshal(map[string]string{
		"order_id":   order.ID.String(),
		"user_id":    userID.String(),
		"payment_id": paymentID.String(),
	})
	if err := s.repo.MarkConfirmed(ctx, order.ID, paymentID, confirmPayload); err != nil {
		return nil, err
	}

	order.Status = domain.OrderConfirmed
	order.PaymentID = &paymentID
	s.logger.Info("order confirmed", "order_id", order.ID, "payment_id", paymentID)

	if b, err := json.Marshal(order); err == nil {
		_ = s.rdb.Set(ctx, idempCacheKey(idempotencyKey), b, idempotencyTTL).Err()
	}
	return order, nil
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

func (s *orderService) ListOrders(ctx context.Context, userID uuid.UUID) ([]*domain.Order, error) {
	return s.repo.GetByUserID(ctx, userID)
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

	// Release any inventory reservations still held.
	items, err := s.repo.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	s.compensate(ctx, orderID, items)

	cancelPayload, _ := json.Marshal(map[string]string{
		"order_id": orderID.String(),
		"user_id":  userID.String(),
		"reason":   "user_requested",
	})
	if err := s.repo.MarkCancelled(ctx, orderID, cancelPayload); err != nil {
		return nil, err
	}

	order.Status = domain.OrderCancelled
	s.logger.Info("order cancelled by user", "order_id", orderID)
	return order, nil
}

// compensate releases all reservations that were already made before a
// failure. Errors are logged but not returned — the caller's error takes
// precedence. A failed release here means stock is temporarily stranded in
// RESERVED state; it will be recovered by the TTL sweep job (Stage 12).
func (s *orderService) compensate(ctx context.Context, orderID uuid.UUID, reserved []*domain.OrderItem) {
	for _, item := range reserved {
		if item.ReservationID == nil {
			continue
		}
		if err := s.inventory.ReleaseStock(ctx, *item.ReservationID); err != nil {
			s.logger.Error("compensation: failed to release reservation",
				"order_id", orderID,
				"reservation_id", item.ReservationID,
				"error", err,
			)
		}
	}
}
