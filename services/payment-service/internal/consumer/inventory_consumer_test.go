package consumer_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"io"
	"log/slog"

	"github.com/google/uuid"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/consumer"
	"github.com/zapmarket/zapmarket/services/payment-service/internal/domain"
)

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeCardCharger struct {
	payment   *domain.Payment
	chargeErr error
}

func (f *fakeCardCharger) ChargeCard(
	ctx context.Context,
	orderID, userID uuid.UUID,
	amount int64, currency string,
	idempotencyKey uuid.UUID,
	paymentMethodID string,
) (*domain.Payment, error) {
	if f.chargeErr != nil {
		return nil, f.chargeErr
	}
	return f.payment, nil
}

type fakePublisher struct {
	msgs []pkgkafka.Message
	err  error
}

func (f *fakePublisher) Publish(_ context.Context, msg pkgkafka.Message) error {
	if f.err != nil {
		return f.err
	}
	f.msgs = append(f.msgs, msg)
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func makeEvent(orderID, userID string) pkgkafka.Message {
	evt := consumer.InventoryReservedEvent{
		OrderID:         orderID,
		UserID:          userID,
		AmountCents:     9900,
		Currency:        "INR",
		PaymentMethodID: "pm_test_123",
		ReservedAt:      time.Now(),
	}
	b, _ := json.Marshal(evt)
	return pkgkafka.Message{Key: []byte(orderID), Value: b}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestInventoryConsumer_SuccessPublishesToCaptured(t *testing.T) {
	orderID := uuid.New().String()
	userID := uuid.New().String()
	paymentID := uuid.New()

	svc := &fakeCardCharger{
		payment: &domain.Payment{
			ID:     paymentID,
			Status: domain.PaymentCaptured,
		},
	}
	capturedPub := &fakePublisher{}
	failedPub := &fakePublisher{}

	c := consumer.NewInventoryConsumer(svc, capturedPub, failedPub, discardLogger())
	if err := c.Handle(context.Background(), makeEvent(orderID, userID)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedPub.msgs) != 1 {
		t.Fatalf("expected 1 captured message, got %d", len(capturedPub.msgs))
	}
	if len(failedPub.msgs) != 0 {
		t.Fatalf("expected 0 failed messages, got %d", len(failedPub.msgs))
	}

	// Verify captured event contains payment ID and order ID.
	var captured consumer.PaymentCapturedEvent
	if err := json.Unmarshal(capturedPub.msgs[0].Value, &captured); err != nil {
		t.Fatalf("unmarshal PaymentCapturedEvent: %v", err)
	}
	if captured.OrderID != orderID {
		t.Errorf("order_id mismatch: got %q want %q", captured.OrderID, orderID)
	}
	if captured.PaymentID != paymentID.String() {
		t.Errorf("payment_id mismatch: got %q want %q", captured.PaymentID, paymentID.String())
	}
}

func TestInventoryConsumer_ChargeErrorPublishesToFailed(t *testing.T) {
	orderID := uuid.New().String()
	userID := uuid.New().String()

	svc := &fakeCardCharger{chargeErr: errors.New("card declined")}
	capturedPub := &fakePublisher{}
	failedPub := &fakePublisher{}

	c := consumer.NewInventoryConsumer(svc, capturedPub, failedPub, discardLogger())
	if err := c.Handle(context.Background(), makeEvent(orderID, userID)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(failedPub.msgs) != 1 {
		t.Fatalf("expected 1 failed message, got %d", len(failedPub.msgs))
	}
	if len(capturedPub.msgs) != 0 {
		t.Fatalf("expected 0 captured messages, got %d", len(capturedPub.msgs))
	}

	var failed consumer.PaymentFailedEvent
	if err := json.Unmarshal(failedPub.msgs[0].Value, &failed); err != nil {
		t.Fatalf("unmarshal PaymentFailedEvent: %v", err)
	}
	if failed.OrderID != orderID {
		t.Errorf("order_id mismatch: got %q want %q", failed.OrderID, orderID)
	}
	if failed.Reason != "card declined" {
		t.Errorf("reason mismatch: got %q want %q", failed.Reason, "card declined")
	}
}

func TestInventoryConsumer_GatewayDeclinePublishesToFailed(t *testing.T) {
	orderID := uuid.New().String()
	userID := uuid.New().String()
	reason := "insufficient funds"

	svc := &fakeCardCharger{
		payment: &domain.Payment{
			ID:            uuid.New(),
			Status:        domain.PaymentFailed,
			FailureReason: &reason,
		},
	}
	capturedPub := &fakePublisher{}
	failedPub := &fakePublisher{}

	c := consumer.NewInventoryConsumer(svc, capturedPub, failedPub, discardLogger())
	if err := c.Handle(context.Background(), makeEvent(orderID, userID)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(failedPub.msgs) != 1 {
		t.Fatalf("expected 1 failed message, got %d", len(failedPub.msgs))
	}
}

func TestInventoryConsumer_MalformedMessageIsSkipped(t *testing.T) {
	svc := &fakeCardCharger{}
	capturedPub := &fakePublisher{}
	failedPub := &fakePublisher{}

	c := consumer.NewInventoryConsumer(svc, capturedPub, failedPub, discardLogger())
	msg := pkgkafka.Message{Value: []byte("not-json")}
	// Should not return an error (malformed messages are skipped, not retried).
	if err := c.Handle(context.Background(), msg); err != nil {
		t.Fatalf("expected nil error for malformed message, got %v", err)
	}
}
