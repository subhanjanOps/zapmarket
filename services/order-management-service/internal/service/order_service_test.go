package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain/contracts"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/infrastructure/cache"
)

// ── mock repository ──────────────────────────────────────────────────────────

type mockOrderRepo struct {
	getByIdempotencyKeyFn func(ctx context.Context, key uuid.UUID) (*domain.Order, error)
	createOrderFn         func(ctx context.Context, order *domain.Order, items []*domain.OrderItem) error
	getByIDFn             func(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	getOrderItemsFn       func(ctx context.Context, orderID uuid.UUID) ([]*domain.OrderItem, error)
	getByUserIDFn         func(ctx context.Context, userID uuid.UUID, p contracts.OrderPageParams) ([]*domain.Order, int64, error)
	getBySellerIDFn       func(ctx context.Context, sellerID uuid.UUID, p contracts.OrderPageParams) ([]*domain.Order, int64, error)
	markReservedFn        func(ctx context.Context, orderID uuid.UUID, items []*domain.OrderItem) error
	markConfirmedFn       func(ctx context.Context, orderID uuid.UUID, paymentID uuid.UUID, payload []byte) error
	markCancelledFn       func(ctx context.Context, orderID uuid.UUID, payload []byte) error
	listAllFn             func(ctx context.Context, params contracts.OrderListParams) ([]*domain.Order, int64, error)
}

func (m *mockOrderRepo) GetByIdempotencyKey(ctx context.Context, key uuid.UUID) (*domain.Order, error) {
	if m.getByIdempotencyKeyFn != nil {
		return m.getByIdempotencyKeyFn(ctx, key)
	}
	return nil, pkgerrors.NewNotFound("ORDER_NOT_FOUND", "not found")
}
func (m *mockOrderRepo) CreateOrder(ctx context.Context, order *domain.Order, items []*domain.OrderItem) error {
	if m.createOrderFn != nil {
		return m.createOrderFn(ctx, order, items)
	}
	order.ID = uuid.New()
	return nil
}
func (m *mockOrderRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, pkgerrors.NewNotFound("ORDER_NOT_FOUND", "not found")
}
func (m *mockOrderRepo) GetOrderItems(ctx context.Context, orderID uuid.UUID) ([]*domain.OrderItem, error) {
	if m.getOrderItemsFn != nil {
		return m.getOrderItemsFn(ctx, orderID)
	}
	return nil, nil
}
func (m *mockOrderRepo) GetByUserID(ctx context.Context, userID uuid.UUID, p contracts.OrderPageParams) ([]*domain.Order, int64, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID, p)
	}
	return nil, 0, nil
}
func (m *mockOrderRepo) GetBySellerID(ctx context.Context, sellerID uuid.UUID, p contracts.OrderPageParams) ([]*domain.Order, int64, error) {
	if m.getBySellerIDFn != nil {
		return m.getBySellerIDFn(ctx, sellerID, p)
	}
	return nil, 0, nil
}
func (m *mockOrderRepo) MarkReserved(ctx context.Context, orderID uuid.UUID, items []*domain.OrderItem) error {
	if m.markReservedFn != nil {
		return m.markReservedFn(ctx, orderID, items)
	}
	return nil
}
func (m *mockOrderRepo) MarkConfirmed(ctx context.Context, orderID uuid.UUID, paymentID uuid.UUID, payload []byte) error {
	if m.markConfirmedFn != nil {
		return m.markConfirmedFn(ctx, orderID, paymentID, payload)
	}
	return nil
}
func (m *mockOrderRepo) MarkCancelled(ctx context.Context, orderID uuid.UUID, payload []byte) error {
	if m.markCancelledFn != nil {
		return m.markCancelledFn(ctx, orderID, payload)
	}
	return nil
}
func (m *mockOrderRepo) ListAll(ctx context.Context, params contracts.OrderListParams) ([]*domain.Order, int64, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx, params)
	}
	return nil, 0, nil
}

// ── mock clients ─────────────────────────────────────────────────────────────

type mockInventory struct {
	reserveFn func(ctx context.Context, skuID, orderID uuid.UUID, qty int) (uuid.UUID, bool, error)
	releaseFn func(ctx context.Context, reservationID uuid.UUID) error
	deductFn  func(ctx context.Context, reservationID uuid.UUID) error
}

func (m *mockInventory) ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (uuid.UUID, bool, error) {
	if m.reserveFn != nil {
		return m.reserveFn(ctx, skuID, orderID, qty)
	}
	return uuid.New(), true, nil
}
func (m *mockInventory) ReleaseStock(ctx context.Context, reservationID uuid.UUID) error {
	if m.releaseFn != nil {
		return m.releaseFn(ctx, reservationID)
	}
	return nil
}
func (m *mockInventory) DeductStock(ctx context.Context, reservationID uuid.UUID) error {
	if m.deductFn != nil {
		return m.deductFn(ctx, reservationID)
	}
	return nil
}

type mockPayment struct {
	chargeFn func(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, idempotencyKey uuid.UUID) (uuid.UUID, string, error)
}

func (m *mockPayment) ChargeCard(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, idempotencyKey uuid.UUID) (uuid.UUID, string, error) {
	if m.chargeFn != nil {
		return m.chargeFn(ctx, orderID, userID, amount, currency, idempotencyKey)
	}
	return uuid.New(), "CAPTURED", nil
}

type mockCatalog struct {
	getPriceFn func(ctx context.Context, skuID uuid.UUID) (int64, error)
}

func (m *mockCatalog) GetSKUPrice(ctx context.Context, skuID uuid.UUID) (int64, error) {
	if m.getPriceFn != nil {
		return m.getPriceFn(ctx, skuID)
	}
	return 500, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func newTestService(t *testing.T, repo contracts.OrderRepository, inv inventoryGateway, pay paymentGateway, rdb *redis.Client) OrderService {
	t.Helper()
	return NewOrderService(repo, inv, pay, &mockCatalog{}, cache.NewRedisCache(rdb), slog.Default())
}

func defaultItems() []CheckoutItem {
	return []CheckoutItem{{
		SKUID:    uuid.New(),
		Quantity: 2,
	}}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCheckout_HappyPath(t *testing.T) {
	rdb := newTestRedis(t)
	repo := &mockOrderRepo{}
	inv := &mockInventory{}
	pay := &mockPayment{}

	svc := newTestService(t, repo, inv, pay, rdb)

	userID := uuid.New()
	idemKey := uuid.New()

	order, err := svc.Checkout(context.Background(), userID, idemKey, defaultItems(), "INR")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if order.Status != domain.OrderConfirmed {
		t.Errorf("expected CONFIRMED, got %s", order.Status)
	}
	if order.PaymentID == nil {
		t.Error("expected PaymentID to be set")
	}
}

func TestCheckout_ValidationErrors(t *testing.T) {
	rdb := newTestRedis(t)
	repo := &mockOrderRepo{}
	inv := &mockInventory{}
	pay := &mockPayment{}
	svc := newTestService(t, repo, inv, pay, rdb)
	ctx := context.Background()

	t.Run("nil user ID", func(t *testing.T) {
		_, err := svc.Checkout(ctx, uuid.Nil, uuid.New(), defaultItems(), "INR")
		assertValidationError(t, err)
	})
	t.Run("nil idempotency key", func(t *testing.T) {
		_, err := svc.Checkout(ctx, uuid.New(), uuid.Nil, defaultItems(), "INR")
		assertValidationError(t, err)
	})
	t.Run("empty items", func(t *testing.T) {
		_, err := svc.Checkout(ctx, uuid.New(), uuid.New(), nil, "INR")
		assertValidationError(t, err)
	})
	t.Run("zero quantity", func(t *testing.T) {
		items := []CheckoutItem{{SKUID: uuid.New(), Quantity: 0}}
		_, err := svc.Checkout(ctx, uuid.New(), uuid.New(), items, "INR")
		assertValidationError(t, err)
	})
	// unit_price is now fetched from catalog; client-supplied value is ignored
}

func TestCheckout_InventoryFailure(t *testing.T) {
	rdb := newTestRedis(t)
	repo := &mockOrderRepo{}

	releaseCalled := false
	inv := &mockInventory{
		reserveFn: func(ctx context.Context, skuID, orderID uuid.UUID, qty int) (uuid.UUID, bool, error) {
			return uuid.Nil, false, nil // insufficient stock
		},
		releaseFn: func(ctx context.Context, reservationID uuid.UUID) error {
			releaseCalled = true
			return nil
		},
	}
	pay := &mockPayment{}
	svc := newTestService(t, repo, inv, pay, rdb)

	_, err := svc.Checkout(context.Background(), uuid.New(), uuid.New(), defaultItems(), "INR")
	if err == nil {
		t.Fatal("expected error from insufficient stock")
	}
	if !isConflictError(err, "INSUFFICIENT_STOCK") {
		t.Errorf("expected INSUFFICIENT_STOCK conflict, got %v", err)
	}
	// No reservations were made, so release must not be called.
	if releaseCalled {
		t.Error("release should not be called when reserve returned false")
	}
}

func TestCheckout_PaymentFailure_ReleasesReservations(t *testing.T) {
	rdb := newTestRedis(t)
	reservationID := uuid.New()
	releaseCount := 0

	repo := &mockOrderRepo{}
	inv := &mockInventory{
		reserveFn: func(ctx context.Context, skuID, orderID uuid.UUID, qty int) (uuid.UUID, bool, error) {
			return reservationID, true, nil
		},
		releaseFn: func(ctx context.Context, rid uuid.UUID) error {
			if rid == reservationID {
				releaseCount++
			}
			return nil
		},
	}
	pay := &mockPayment{
		chargeFn: func(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, key uuid.UUID) (uuid.UUID, string, error) {
			return uuid.Nil, "", pkgerrors.NewInternal("PAYMENT_ERROR", "gateway timeout", nil)
		},
	}
	svc := newTestService(t, repo, inv, pay, rdb)

	_, err := svc.Checkout(context.Background(), uuid.New(), uuid.New(), defaultItems(), "INR")
	if err == nil {
		t.Fatal("expected payment error")
	}
	if releaseCount != 1 {
		t.Errorf("expected 1 release call, got %d", releaseCount)
	}
}

func TestCheckout_PaymentNotCaptured(t *testing.T) {
	rdb := newTestRedis(t)
	releaseCount := 0

	repo := &mockOrderRepo{}
	inv := &mockInventory{
		reserveFn: func(ctx context.Context, skuID, orderID uuid.UUID, qty int) (uuid.UUID, bool, error) {
			return uuid.New(), true, nil
		},
		releaseFn: func(ctx context.Context, rid uuid.UUID) error {
			releaseCount++
			return nil
		},
	}
	pay := &mockPayment{
		chargeFn: func(ctx context.Context, orderID, userID uuid.UUID, amount int64, currency string, key uuid.UUID) (uuid.UUID, string, error) {
			return uuid.New(), "FAILED", nil
		},
	}
	svc := newTestService(t, repo, inv, pay, rdb)

	_, err := svc.Checkout(context.Background(), uuid.New(), uuid.New(), defaultItems(), "INR")
	if err == nil {
		t.Fatal("expected payment-not-captured error")
	}
	if !isConflictError(err, "PAYMENT_FAILED") {
		t.Errorf("expected PAYMENT_FAILED, got %v", err)
	}
	if releaseCount != 1 {
		t.Errorf("expected reservation released, got %d release calls", releaseCount)
	}
}

func TestCheckout_IdempotencyReplay(t *testing.T) {
	rdb := newTestRedis(t)
	idemKey := uuid.New()

	existing := &domain.Order{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		IdempotencyKey: idemKey,
		Status:         domain.OrderConfirmed,
		TotalAmount:    1000,
		Currency:       "INR",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Pre-warm the cache the way the real service does.
	b, _ := json.Marshal(existing)
	_ = rdb.Set(context.Background(), idempCacheKey(idemKey), b, idempotencyTTL).Err()

	createCalled := false
	repo := &mockOrderRepo{
		createOrderFn: func(ctx context.Context, order *domain.Order, items []*domain.OrderItem) error {
			createCalled = true
			return nil
		},
	}
	inv := &mockInventory{}
	pay := &mockPayment{}
	svc := newTestService(t, repo, inv, pay, rdb)

	order, err := svc.Checkout(context.Background(), existing.UserID, idemKey, defaultItems(), "INR")
	if err != nil {
		t.Fatalf("idempotent replay returned error: %v", err)
	}
	if order.ID != existing.ID {
		t.Errorf("expected replay to return original order %s, got %s", existing.ID, order.ID)
	}
	if createCalled {
		t.Error("CreateOrder must not be called on idempotent replay")
	}
}

func TestCheckout_IdempotencyReplay_DBFallback(t *testing.T) {
	rdb := newTestRedis(t)
	idemKey := uuid.New()

	existing := &domain.Order{
		ID:             uuid.New(),
		IdempotencyKey: idemKey,
		Status:         domain.OrderConfirmed,
		TotalAmount:    1000,
		Currency:       "INR",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	createCalled := false
	repo := &mockOrderRepo{
		getByIdempotencyKeyFn: func(ctx context.Context, key uuid.UUID) (*domain.Order, error) {
			return existing, nil
		},
		createOrderFn: func(ctx context.Context, order *domain.Order, items []*domain.OrderItem) error {
			createCalled = true
			return nil
		},
	}
	inv := &mockInventory{}
	pay := &mockPayment{}
	svc := newTestService(t, repo, inv, pay, rdb)

	order, err := svc.Checkout(context.Background(), uuid.New(), idemKey, defaultItems(), "INR")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if order.ID != existing.ID {
		t.Errorf("expected DB-replayed order %s, got %s", existing.ID, order.ID)
	}
	if createCalled {
		t.Error("CreateOrder must not be called on idempotent DB replay")
	}
}

// ── assertion helpers ─────────────────────────────────────────────────────────

func assertValidationError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var appErr *pkgerrors.AppError
	if !isAppError(err, &appErr) || appErr.Type != pkgerrors.Validation {
		t.Errorf("expected Validation AppError, got %T: %v", err, err)
	}
}

func isConflictError(err error, code string) bool {
	var appErr *pkgerrors.AppError
	if !isAppError(err, &appErr) {
		return false
	}
	return appErr.Type == pkgerrors.Conflict && appErr.Code == code
}

func isAppError(err error, target **pkgerrors.AppError) bool {
	if err == nil {
		return false
	}
	ae, ok := err.(*pkgerrors.AppError)
	if ok && target != nil {
		*target = ae
	}
	return ok
}
