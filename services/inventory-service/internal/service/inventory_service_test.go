package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain/contracts"
)

// ── mock repository ──────────────────────────────────────────────────────────

type mockInventoryRepo struct {
	addStockFn              func(ctx context.Context, skuID uuid.UUID, qty int) (int, error)
	reserveStockFn          func(ctx context.Context, skuID, orderID uuid.UUID, qty int) (*domain.Reservation, error)
	releaseStockFn          func(ctx context.Context, reservationID uuid.UUID) (uuid.UUID, int64, error)
	deductStockFn           func(ctx context.Context, reservationID uuid.UUID) error
	getStockFn              func(ctx context.Context, skuID uuid.UUID) (*domain.Inventory, error)
	getReservationDetailsFn func(ctx context.Context, reservationID uuid.UUID) (uuid.UUID, int, error)
}

func (m *mockInventoryRepo) AddStock(ctx context.Context, skuID uuid.UUID, qty int) (int, error) {
	if m.addStockFn != nil {
		return m.addStockFn(ctx, skuID, qty)
	}
	return qty, nil
}

func (m *mockInventoryRepo) ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (*domain.Reservation, error) {
	if m.reserveStockFn != nil {
		return m.reserveStockFn(ctx, skuID, orderID, qty)
	}
	return &domain.Reservation{ID: uuid.New(), SKUID: skuID, OrderID: orderID, Qty: qty}, nil
}

func (m *mockInventoryRepo) ReleaseStock(ctx context.Context, reservationID uuid.UUID) (uuid.UUID, int64, error) {
	if m.releaseStockFn != nil {
		return m.releaseStockFn(ctx, reservationID)
	}
	return uuid.New(), 1, nil
}

func (m *mockInventoryRepo) DeductStock(ctx context.Context, reservationID uuid.UUID) error {
	if m.deductStockFn != nil {
		return m.deductStockFn(ctx, reservationID)
	}
	return nil
}

func (m *mockInventoryRepo) GetStock(ctx context.Context, skuID uuid.UUID) (*domain.Inventory, error) {
	if m.getStockFn != nil {
		return m.getStockFn(ctx, skuID)
	}
	return &domain.Inventory{SKUID: skuID, QtyAvailable: 100}, nil
}

func (m *mockInventoryRepo) GetReservationDetails(ctx context.Context, reservationID uuid.UUID) (uuid.UUID, int, error) {
	if m.getReservationDetailsFn != nil {
		return m.getReservationDetailsFn(ctx, reservationID)
	}
	return uuid.New(), 1, nil
}

var _ contracts.InventoryRepository = (*mockInventoryRepo)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func newRealRedis(t *testing.T) *goredis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}

func newDownRedis(t *testing.T) *goredis.Client {
	t.Helper()
	// Point at a port nothing is listening on — every call will fail.
	return goredis.NewClient(&goredis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 1,
	})
}

func newSvc(repo contracts.InventoryRepository, rdb *goredis.Client) InventoryService {
	return NewInventoryService(repo, rdb, slog.Default())
}

// seedRedis sets the stock key directly so the Lua script sees a warm cache.
func seedRedis(t *testing.T, rdb *goredis.Client, skuID uuid.UUID, qty int) {
	t.Helper()
	_ = rdb.Set(context.Background(), stockKey(skuID), qty, 0).Err()
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestReserveStock_RedisHit_Success(t *testing.T) {
	skuID := uuid.New()
	rdb := newRealRedis(t)
	seedRedis(t, rdb, skuID, 10)

	dbCalled := false
	repo := &mockInventoryRepo{
		reserveStockFn: func(ctx context.Context, sid, oid uuid.UUID, qty int) (*domain.Reservation, error) {
			dbCalled = true
			return &domain.Reservation{ID: uuid.New(), SKUID: sid, Qty: qty}, nil
		},
	}
	svc := newSvc(repo, rdb)

	res, err := svc.ReserveStock(context.Background(), skuID, uuid.New(), 3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res == nil {
		t.Fatal("expected a reservation")
	}
	if !dbCalled {
		t.Error("expected DB to be called after Redis gate passed")
	}
	// Redis should have been decremented.
	val, _ := rdb.Get(context.Background(), stockKey(skuID)).Int()
	if val != 7 {
		t.Errorf("expected Redis counter 7 after reserving 3 from 10, got %d", val)
	}
}

func TestReserveStock_RedisMiss_WarmsAndSucceeds(t *testing.T) {
	skuID := uuid.New()
	rdb := newRealRedis(t)
	// Do NOT seed Redis — force a cache miss (result == -1 from Lua).

	repo := &mockInventoryRepo{
		getStockFn: func(ctx context.Context, sid uuid.UUID) (*domain.Inventory, error) {
			return &domain.Inventory{SKUID: sid, QtyAvailable: 5}, nil
		},
		reserveStockFn: func(ctx context.Context, sid, oid uuid.UUID, qty int) (*domain.Reservation, error) {
			return &domain.Reservation{ID: uuid.New(), SKUID: sid, Qty: qty}, nil
		},
	}
	svc := newSvc(repo, rdb)

	res, err := svc.ReserveStock(context.Background(), skuID, uuid.New(), 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res == nil {
		t.Fatal("expected reservation after cache warm + retry")
	}
	// Redis should now be warmed and decremented.
	val, _ := rdb.Get(context.Background(), stockKey(skuID)).Int()
	if val != 3 {
		t.Errorf("expected Redis counter 3 after warming at 5 and reserving 2, got %d", val)
	}
}

func TestReserveStock_InsufficientStock(t *testing.T) {
	skuID := uuid.New()
	rdb := newRealRedis(t)
	seedRedis(t, rdb, skuID, 1)

	dbCalled := false
	repo := &mockInventoryRepo{
		reserveStockFn: func(ctx context.Context, sid, oid uuid.UUID, qty int) (*domain.Reservation, error) {
			dbCalled = true
			return nil, nil
		},
	}
	svc := newSvc(repo, rdb)

	_, err := svc.ReserveStock(context.Background(), skuID, uuid.New(), 5)
	if err == nil {
		t.Fatal("expected insufficient-stock error")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != "INSUFFICIENT_STOCK" {
		t.Errorf("expected INSUFFICIENT_STOCK conflict, got %v", err)
	}
	if dbCalled {
		t.Error("DB must not be called when Redis says insufficient stock")
	}
}

func TestReserveStock_RedisDown_DBFallback(t *testing.T) {
	skuID := uuid.New()
	rdb := newDownRedis(t)

	dbCalled := false
	repo := &mockInventoryRepo{
		reserveStockFn: func(ctx context.Context, sid, oid uuid.UUID, qty int) (*domain.Reservation, error) {
			dbCalled = true
			return &domain.Reservation{ID: uuid.New(), SKUID: sid, Qty: qty}, nil
		},
	}
	svc := newSvc(repo, rdb)

	res, err := svc.ReserveStock(context.Background(), skuID, uuid.New(), 2)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}
	if res == nil {
		t.Fatal("expected reservation from DB fallback")
	}
	if !dbCalled {
		t.Error("DB fallback must be called when Redis is down")
	}
}

func TestReserveStock_DBRejectsAfterLuaPass_RedisRolledBack(t *testing.T) {
	skuID := uuid.New()
	rdb := newRealRedis(t)
	seedRedis(t, rdb, skuID, 10)

	dbErr := errors.New("db constraint violation")
	repo := &mockInventoryRepo{
		reserveStockFn: func(ctx context.Context, sid, oid uuid.UUID, qty int) (*domain.Reservation, error) {
			return nil, dbErr
		},
	}
	svc := newSvc(repo, rdb)

	_, err := svc.ReserveStock(context.Background(), skuID, uuid.New(), 3)
	if err == nil {
		t.Fatal("expected error from DB failure")
	}

	// Redis counter must be rolled back to 10 (decremented by 3, then incremented back by 3).
	val, _ := rdb.Get(context.Background(), stockKey(skuID)).Int()
	if val != 10 {
		t.Errorf("expected Redis counter 10 after rollback, got %d", val)
	}
}

func TestReserveStock_ValidationErrors(t *testing.T) {
	rdb := newRealRedis(t)
	repo := &mockInventoryRepo{}
	svc := newSvc(repo, rdb)
	ctx := context.Background()

	t.Run("nil sku_id", func(t *testing.T) {
		_, err := svc.ReserveStock(ctx, uuid.Nil, uuid.New(), 1)
		assertValidation(t, err)
	})
	t.Run("nil order_id", func(t *testing.T) {
		_, err := svc.ReserveStock(ctx, uuid.New(), uuid.Nil, 1)
		assertValidation(t, err)
	})
	t.Run("zero quantity", func(t *testing.T) {
		_, err := svc.ReserveStock(ctx, uuid.New(), uuid.New(), 0)
		assertValidation(t, err)
	})
}

func assertValidation(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Type != pkgerrors.Validation {
		t.Errorf("expected Validation AppError, got %T: %v", err, err)
	}
}
