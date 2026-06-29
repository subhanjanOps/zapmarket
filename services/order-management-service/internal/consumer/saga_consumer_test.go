package consumer_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/consumer"
	"github.com/zapmarket/zapmarket/services/order-management-service/internal/domain/events"
)

type fakeOrderRepo struct {
	updatedStatus     string
	updatedSagaStatus string
}

func (f *fakeOrderRepo) UpdateStatus(_ context.Context, _ uuid.UUID, status, sagaStatus string) error {
	f.updatedStatus = status
	f.updatedSagaStatus = sagaStatus
	return nil
}

func (f *fakeOrderRepo) SetItemReservationID(_ context.Context, _, _, _ uuid.UUID) error {
	return nil
}

type fakeCompensator struct {
	releasedOrderID string
	refundedOrderID string
}

func (f *fakeCompensator) ReleaseStock(_ context.Context, orderID string) error {
	f.releasedOrderID = orderID
	return nil
}

func (f *fakeCompensator) RefundPayment(_ context.Context, orderID string) error {
	f.refundedOrderID = orderID
	return nil
}

type fakePublisher struct{ topic string }

func (f *fakePublisher) Publish(_ context.Context, topic, _ string, _ []byte) error {
	f.topic = topic
	return nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func makeMsg(v any) kafka.Message {
	b, _ := json.Marshal(v)
	return kafka.Message{Value: b}
}

func TestSagaConsumer_ConfirmsOrderOnPaymentCaptured(t *testing.T) {
	repo := &fakeOrderRepo{}
	comp := &fakeCompensator{}
	pub := &fakePublisher{}
	c := consumer.NewSagaConsumer(repo, comp, pub, newTestLogger())

	orderID := uuid.New().String()
	evt := events.PaymentCapturedEvent{OrderID: orderID, PaymentID: "pay-1", CapturedAt: time.Now()}
	if err := c.HandlePaymentCaptured(context.Background(), makeMsg(evt)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updatedStatus != "CONFIRMED" {
		t.Fatalf("expected CONFIRMED, got %q", repo.updatedStatus)
	}
	if repo.updatedSagaStatus != "COMPLETE" {
		t.Fatalf("expected COMPLETE, got %q", repo.updatedSagaStatus)
	}
	if pub.topic != kafka.TopicOrderConfirmed {
		t.Fatalf("expected %q, got %q", kafka.TopicOrderConfirmed, pub.topic)
	}
}

func TestSagaConsumer_CancelsOrderAndNoReleaseOnInventoryFailed(t *testing.T) {
	repo := &fakeOrderRepo{}
	comp := &fakeCompensator{}
	pub := &fakePublisher{}
	c := consumer.NewSagaConsumer(repo, comp, pub, newTestLogger())

	orderID := uuid.New().String()
	evt := events.InventoryReservationFailedEvent{OrderID: orderID, Reason: "out of stock"}
	if err := c.HandleInventoryFailed(context.Background(), makeMsg(evt)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updatedStatus != "CANCELLED" {
		t.Fatalf("expected CANCELLED, got %q", repo.updatedStatus)
	}
	// inventory never reserved — compensator should NOT have been called
	if comp.releasedOrderID != "" {
		t.Fatalf("release should not be called when inventory never reserved, got %q", comp.releasedOrderID)
	}
	if pub.topic != kafka.TopicOrderCancelled {
		t.Fatalf("expected %q, got %q", kafka.TopicOrderCancelled, pub.topic)
	}
}

func TestSagaConsumer_CancelsAndReleasesOnPaymentFailed(t *testing.T) {
	repo := &fakeOrderRepo{}
	comp := &fakeCompensator{}
	pub := &fakePublisher{}
	c := consumer.NewSagaConsumer(repo, comp, pub, newTestLogger())

	orderID := uuid.New().String()
	evt := events.PaymentFailedEvent{OrderID: orderID, Reason: "card declined"}
	if err := c.HandlePaymentFailed(context.Background(), makeMsg(evt)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updatedStatus != "CANCELLED" {
		t.Fatalf("expected CANCELLED, got %q", repo.updatedStatus)
	}
	if comp.releasedOrderID != orderID {
		t.Fatalf("expected inventory release for %s, got %q", orderID, comp.releasedOrderID)
	}
	if pub.topic != kafka.TopicOrderCancelled {
		t.Fatalf("expected %q, got %q", kafka.TopicOrderCancelled, pub.topic)
	}
}

func TestSagaConsumer_SkipsInvalidOrderID(t *testing.T) {
	repo := &fakeOrderRepo{}
	comp := &fakeCompensator{}
	pub := &fakePublisher{}
	c := consumer.NewSagaConsumer(repo, comp, pub, newTestLogger())

	evt := events.PaymentCapturedEvent{OrderID: "not-a-uuid", PaymentID: "pay-1", CapturedAt: time.Now()}
	if err := c.HandlePaymentCaptured(context.Background(), makeMsg(evt)); err != nil {
		t.Fatalf("expected nil error for invalid UUID, got %v", err)
	}
	if repo.updatedStatus != "" {
		t.Fatal("UpdateStatus should not be called for an invalid order_id")
	}
}
