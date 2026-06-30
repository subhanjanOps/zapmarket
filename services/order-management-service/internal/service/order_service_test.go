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
	createOrderFn         func(ctx context.Context, order *domain.Order, items []*domain.OrderItem, outboxPayload []byte) error
	getByIDFn             func(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	getOrderItemsFn       func(ctx context.Context, orderID uuid.UUID) ([]*domain.OrderItem, error)
	getByUserIDFn         func(ctx context.Context, userID uuid.UUID, p contracts.OrderPageParams) ([]*domain.Order, int64, error)
	getBySellerIDFn       func(ctx context.Context, sellerID uuid.UUID, p contracts.OrderPageParams) ([]*domain.Order, int64, error)
	markReservedFn        func(ctx context.Context, orderID uuid.UUID, items []*domain.OrderItem) error
	markConfirmedFn       func(ctx context.Context, orderID uuid.UUID, paymentID uuid.UUID) error
	markCancelledFn       func(ctx context.Context, orderID uuid.UUID) error
	listAllFn             func(ctx context.Context, params contracts.OrderListParams) ([]*domain.Order, int64, error)
}

func (m *mockOrderRepo) GetByIdempotencyKey(ctx context.Context, key uuid.UUID) (*domain.Order, error) {
	if m.getByIdempotencyKeyFn != nil {
		return m.getByIdempotencyKeyFn(ctx, key)
	}
	return nil, pkgerrors.NewNotFound("ORDER_NOT_FOUND", "not found")
}
func (m *mockOrderRepo) CreateOrder(ctx context.Context, order *domain.Order, items []*domain.OrderItem, outboxPayload []byte) error {
	if m.createOrderFn != nil {
		return m.createOrderFn(ctx, order, items, outboxPayload)
	}
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
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
func (m *mockOrderRepo) MarkConfirmed(ctx context.Context, orderID uuid.UUID, paymentID uuid.UUID) error {
	if m.markConfirmedFn != nil {
		return m.markConfirmedFn(ctx, orderID, paymentID)
	}
	return nil
}
func (m *mockOrderRepo) MarkCancelled(ctx context.Context, orderID uuid.UUID) error {
	if m.markCancelledFn != nil {
		return m.markCancelledFn(ctx, orderID)
	}
	return nil
}
func (m *mockOrderRepo) SetItemReservationID(_ context.Context, _, _, _ uuid.UUID) error {
	return nil
}
func (m *mockOrderRepo) ListAll(ctx context.Context, params contracts.OrderListParams) ([]*domain.Order, int64, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx, params)
	}
	return nil, 0, nil
}
func (m *mockOrderRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _, _ string) error { return nil }

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

func newTestService(t *testing.T, repo contracts.OrderRepository, inv inventoryGateway, rdb *redis.Client) OrderService {
	t.Helper()
	return NewOrderService(repo, inv, &mockCatalog{}, nil, cache.NewRedisCache(rdb), slog.Default())
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

	svc := NewOrderService(repo, inv, &mockCatalog{}, nil, cache.NewRedisCache(rdb), slog.Default())

	userID := uuid.New()
	idemKey := uuid.New()

	order, err := svc.Checkout(context.Background(), userID, idemKey, defaultItems(), "INR", "pm-123", CheckoutOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Checkout now returns PENDING — the saga consumer confirms later.
	if order.Status != domain.OrderPending {
		t.Errorf("expected PENDING, got %s", order.Status)
	}
	if order.SagaStatus != "AWAITING_INVENTORY" {
		t.Errorf("expected AWAITING_INVENTORY, got %s", order.SagaStatus)
	}
	if order.ID == uuid.Nil {
		t.Error("expected order.ID to be set")
	}
}

func TestCheckout_ValidationErrors(t *testing.T) {
	rdb := newTestRedis(t)
	repo := &mockOrderRepo{}
	inv := &mockInventory{}
	svc := newTestService(t, repo, inv, rdb)
	ctx := context.Background()

	t.Run("nil user ID", func(t *testing.T) {
		_, err := svc.Checkout(ctx, uuid.Nil, uuid.New(), defaultItems(), "INR", "", CheckoutOptions{})
		assertValidationError(t, err)
	})
	t.Run("nil idempotency key", func(t *testing.T) {
		_, err := svc.Checkout(ctx, uuid.New(), uuid.Nil, defaultItems(), "INR", "", CheckoutOptions{})
		assertValidationError(t, err)
	})
	t.Run("empty items", func(t *testing.T) {
		_, err := svc.Checkout(ctx, uuid.New(), uuid.New(), nil, "INR", "", CheckoutOptions{})
		assertValidationError(t, err)
	})
	t.Run("zero quantity", func(t *testing.T) {
		items := []CheckoutItem{{SKUID: uuid.New(), Quantity: 0}}
		_, err := svc.Checkout(ctx, uuid.New(), uuid.New(), items, "INR", "", CheckoutOptions{})
		assertValidationError(t, err)
	})
	// unit_price is now fetched from catalog; client-supplied value is ignored
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
		createOrderFn: func(ctx context.Context, order *domain.Order, items []*domain.OrderItem, outboxPayload []byte) error {
			createCalled = true
			return nil
		},
	}
	inv := &mockInventory{}
	svc := newTestService(t, repo, inv, rdb)

	order, err := svc.Checkout(context.Background(), existing.UserID, idemKey, defaultItems(), "INR", "", CheckoutOptions{})
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
		createOrderFn: func(ctx context.Context, order *domain.Order, items []*domain.OrderItem, outboxPayload []byte) error {
			createCalled = true
			return nil
		},
	}
	inv := &mockInventory{}
	svc := newTestService(t, repo, inv, rdb)

	order, err := svc.Checkout(context.Background(), uuid.New(), idemKey, defaultItems(), "INR", "", CheckoutOptions{})
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

func TestCheckout_SetsIdempotencyCacheAfterCreate(t *testing.T) {
	rdb := newTestRedis(t)
	idemKey := uuid.New()
	repo := &mockOrderRepo{}
	svc := newTestService(t, repo, &mockInventory{}, rdb)

	order, err := svc.Checkout(context.Background(), uuid.New(), idemKey, defaultItems(), "INR", "", CheckoutOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The cache should now hold the PENDING order for subsequent duplicate requests.
	cached, err := rdb.Get(context.Background(), idempCacheKey(idemKey)).Bytes()
	if err != nil {
		t.Fatalf("expected cache entry, got error: %v", err)
	}
	var cachedOrder domain.Order
	if err := json.Unmarshal(cached, &cachedOrder); err != nil {
		t.Fatalf("failed to unmarshal cached order: %v", err)
	}
	if cachedOrder.ID != order.ID {
		t.Errorf("cached order ID %s does not match returned order ID %s", cachedOrder.ID, order.ID)
	}
	if cachedOrder.Status != domain.OrderPending {
		t.Errorf("expected PENDING in cache, got %s", cachedOrder.Status)
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
