package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain/contracts"
)

// ── mocks ────────────────────────────────────────────────────────────────────

type mockRepo struct {
	getByIdemFn    func(ctx context.Context, key uuid.UUID) (*domain.Payment, error)
	createFn       func(ctx context.Context, p *domain.Payment) error
	markCapturedFn func(ctx context.Context, id uuid.UUID, txnID string, entries []*domain.LedgerEntry) error
	markFailedFn   func(ctx context.Context, id uuid.UUID, reason string) error
	getByIDFn      func(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	createRefundFn func(ctx context.Context, r *domain.Refund, userID uuid.UUID, status domain.PaymentStatus, entries []*domain.LedgerEntry) error
}

func (m *mockRepo) GetByIdempotencyKey(ctx context.Context, key uuid.UUID) (*domain.Payment, error) {
	if m.getByIdemFn != nil {
		return m.getByIdemFn(ctx, key)
	}
	return nil, pkgerrors.NewNotFound("NOT_FOUND", "not found")
}
func (m *mockRepo) CreatePayment(ctx context.Context, p *domain.Payment) error {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	p.ID = uuid.New()
	return nil
}
func (m *mockRepo) MarkCaptured(ctx context.Context, id uuid.UUID, txnID string, entries []*domain.LedgerEntry) error {
	if m.markCapturedFn != nil {
		return m.markCapturedFn(ctx, id, txnID, entries)
	}
	return nil
}
func (m *mockRepo) MarkFailed(ctx context.Context, id uuid.UUID, reason string) error {
	if m.markFailedFn != nil {
		return m.markFailedFn(ctx, id, reason)
	}
	return nil
}
func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, pkgerrors.NewNotFound("NOT_FOUND", "not found")
}
func (m *mockRepo) CreateRefund(ctx context.Context, r *domain.Refund, userID uuid.UUID, status domain.PaymentStatus, entries []*domain.LedgerEntry) error {
	if m.createRefundFn != nil {
		return m.createRefundFn(ctx, r, userID, status, entries)
	}
	r.ID = uuid.New()
	return nil
}
func (m *mockRepo) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*domain.Payment, error) {
	return nil, pkgerrors.NewNotFound("NOT_FOUND", "not found")
}

func (m *mockRepo) GetByGatewayTxnID(ctx context.Context, txnID string) (*domain.Payment, error) {
	return nil, pkgerrors.NewNotFound("NOT_FOUND", "not found")
}

type mockGateway struct {
	chargeFn func(ctx context.Context, amount int64, currency string, key uuid.UUID, pmID string) (*contracts.ChargeResult, error)
	refundFn func(ctx context.Context, txnID string, amount int64, currency string) (string, error)
}

func (m *mockGateway) Name() string { return "mock" }

func (m *mockGateway) Charge(ctx context.Context, amount int64, currency string, key uuid.UUID, pmID string) (*contracts.ChargeResult, error) {
	if m.chargeFn != nil {
		return m.chargeFn(ctx, amount, currency, key, pmID)
	}
	return &contracts.ChargeResult{GatewayTxnID: "txn_" + uuid.NewString()}, nil
}
func (m *mockGateway) Refund(ctx context.Context, txnID string, amount int64, currency string) (string, error) {
	if m.refundFn != nil {
		return m.refundFn(ctx, txnID, amount, currency)
	}
	return "ref_" + uuid.NewString(), nil
}

type mockCache struct{}

func (m *mockCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, pkgerrors.NewNotFound("MISS", "cache miss")
}
func (m *mockCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error { return nil }
func (m *mockCache) SetNX(_ context.Context, _ string, _ string, _ time.Duration) (bool, error) {
	return true, nil
}
func (m *mockCache) Del(_ context.Context, _ string) error { return nil }

func newSvc(repo contracts.PaymentRepository, gw contracts.PaymentGateway) PaymentService {
	return NewPaymentService(repo, gw, &mockCache{}, slog.Default())
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestChargeCard_HappyPath(t *testing.T) {
	repo := &mockRepo{}
	gw := &mockGateway{}
	svc := newSvc(repo, gw)

	p, err := svc.ChargeCard(context.Background(), uuid.New(), uuid.New(), 1000, "INR", uuid.New(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status != domain.PaymentCaptured {
		t.Errorf("expected CAPTURED, got %s", p.Status)
	}
}

func TestChargeCard_GatewayDecline(t *testing.T) {
	gw := &mockGateway{
		chargeFn: func(_ context.Context, _ int64, _ string, _ uuid.UUID, _ string) (*contracts.ChargeResult, error) {
			return nil, pkgerrors.NewConflict("DECLINED", "card declined")
		},
	}
	svc := newSvc(&mockRepo{}, gw)
	p, err := svc.ChargeCard(context.Background(), uuid.New(), uuid.New(), 500, "INR", uuid.New(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status != domain.PaymentFailed {
		t.Errorf("expected FAILED, got %s", p.Status)
	}
}

func TestChargeCard_IdempotentReplay(t *testing.T) {
	existing := &domain.Payment{ID: uuid.New(), Status: domain.PaymentCaptured, Amount: 999}
	repo := &mockRepo{
		getByIdemFn: func(_ context.Context, _ uuid.UUID) (*domain.Payment, error) {
			return existing, nil
		},
	}
	chargeCount := 0
	gw := &mockGateway{
		chargeFn: func(_ context.Context, _ int64, _ string, _ uuid.UUID, _ string) (*contracts.ChargeResult, error) {
			chargeCount++
			return &contracts.ChargeResult{GatewayTxnID: "txn_x"}, nil
		},
	}
	svc := newSvc(repo, gw)
	p, err := svc.ChargeCard(context.Background(), uuid.New(), uuid.New(), 999, "INR", uuid.New(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != existing.ID {
		t.Error("expected idempotent replay to return existing payment")
	}
	if chargeCount != 0 {
		t.Error("gateway should not be called on replay")
	}
}

func TestChargeCard_ValidationErrors(t *testing.T) {
	svc := newSvc(&mockRepo{}, &mockGateway{})
	ctx := context.Background()

	if _, err := svc.ChargeCard(ctx, uuid.Nil, uuid.New(), 100, "INR", uuid.New(), ""); err == nil {
		t.Error("expected error for nil order_id")
	}
	if _, err := svc.ChargeCard(ctx, uuid.New(), uuid.Nil, 100, "INR", uuid.New(), ""); err == nil {
		t.Error("expected error for nil user_id")
	}
	if _, err := svc.ChargeCard(ctx, uuid.New(), uuid.New(), 0, "INR", uuid.New(), ""); err == nil {
		t.Error("expected error for zero amount")
	}
	if _, err := svc.ChargeCard(ctx, uuid.New(), uuid.New(), 100, "INR", uuid.Nil, ""); err == nil {
		t.Error("expected error for nil idempotency_key")
	}
}
