package consumer_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	pkgkafka "github.com/zapmarket/zapmarket/pkg/kafka"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/consumer"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain"
)

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeInventoryService struct {
	reserveErr     error
	releaseErr     error
	releasedIDs    []uuid.UUID
	reserveResults map[string]*domain.Reservation // key: skuID string
}

func (f *fakeInventoryService) ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (*domain.Reservation, error) {
	if f.reserveErr != nil {
		return nil, f.reserveErr
	}
	res := &domain.Reservation{
		ID:      uuid.New(),
		SKUID:   skuID,
		OrderID: orderID,
		Qty:     qty,
	}
	if f.reserveResults != nil {
		if r, ok := f.reserveResults[skuID.String()]; ok {
			return r, nil
		}
	}
	return res, nil
}

func (f *fakeInventoryService) ReleaseStock(ctx context.Context, reservationID uuid.UUID) error {
	f.releasedIDs = append(f.releasedIDs, reservationID)
	return f.releaseErr
}

type fakePublisher struct {
	published []pkgkafka.Message
}

func (f *fakePublisher) Publish(ctx context.Context, msg pkgkafka.Message) error {
	f.published = append(f.published, msg)
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError + 10}))
}

func makeMessage(evt consumer.CheckoutRequestedEvent) pkgkafka.Message {
	b, _ := json.Marshal(evt)
	return pkgkafka.Message{Key: []byte(evt.OrderID), Value: b}
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestCheckoutConsumer_SuccessfulReservation(t *testing.T) {
	svc := &fakeInventoryService{}
	reservedPub := &fakePublisher{}
	failPub := &fakePublisher{}
	log := discardLogger()

	c := consumer.NewCheckoutConsumer(svc, reservedPub, failPub, log)

	orderID := uuid.New().String()
	skuID := uuid.New().String()
	evt := consumer.CheckoutRequestedEvent{
		OrderID:     orderID,
		UserID:      uuid.New().String(),
		Items:       []consumer.CheckoutItem{{SKUID: skuID, Quantity: 3}},
		RequestedAt: time.Now(),
	}

	if err := c.Handle(context.Background(), makeMessage(evt)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reservedPub.published) != 1 {
		t.Fatalf("expected 1 message on reserved publisher, got %d", len(reservedPub.published))
	}
	if len(failPub.published) != 0 {
		t.Fatalf("expected no messages on failure publisher, got %d", len(failPub.published))
	}

	var out consumer.InventoryReservedEvent
	if err := json.Unmarshal(reservedPub.published[0].Value, &out); err != nil {
		t.Fatalf("could not unmarshal InventoryReservedEvent: %v", err)
	}
	if out.OrderID != orderID {
		t.Errorf("expected OrderID %q, got %q", orderID, out.OrderID)
	}
	if len(out.Reservations) != 1 {
		t.Fatalf("expected 1 reservation ref, got %d", len(out.Reservations))
	}
	if out.Reservations[0].SKUID != skuID {
		t.Errorf("expected SKUID %q, got %q", skuID, out.Reservations[0].SKUID)
	}
	if out.Reservations[0].Quantity != 3 {
		t.Errorf("expected quantity 3, got %d", out.Reservations[0].Quantity)
	}
}

func TestCheckoutConsumer_ReserveFailure_PublishesFailedAndCompensates(t *testing.T) {
	// First item succeeds, second fails — consumer must release the first and
	// publish inventory.reservation_failed.
	firstSKU := uuid.New()
	secondSKU := uuid.New()

	svc2 := &fakeInventorySvcPartial{
		firstSKU: firstSKU,
		firstRes: &domain.Reservation{ID: uuid.New(), SKUID: firstSKU},
		failOn:   secondSKU,
		failErr:  errors.New("insufficient stock"),
	}

	reservedPub := &fakePublisher{}
	failPub := &fakePublisher{}
	log := discardLogger()

	c := consumer.NewCheckoutConsumer(svc2, reservedPub, failPub, log)

	orderID := uuid.New().String()
	evt := consumer.CheckoutRequestedEvent{
		OrderID: orderID,
		Items: []consumer.CheckoutItem{
			{SKUID: firstSKU.String(), Quantity: 1},
			{SKUID: secondSKU.String(), Quantity: 999},
		},
	}

	if err := c.Handle(context.Background(), makeMessage(evt)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must have published to failure topic
	if len(failPub.published) != 1 {
		t.Fatalf("expected 1 message on failure publisher, got %d", len(failPub.published))
	}
	if len(reservedPub.published) != 0 {
		t.Fatalf("expected no messages on reserved publisher, got %d", len(reservedPub.published))
	}

	var out consumer.InventoryReservationFailedEvent
	if err := json.Unmarshal(failPub.published[0].Value, &out); err != nil {
		t.Fatalf("could not unmarshal InventoryReservationFailedEvent: %v", err)
	}
	if out.OrderID != orderID {
		t.Errorf("expected OrderID %q, got %q", orderID, out.OrderID)
	}
	if out.Reason == "" {
		t.Error("expected non-empty reason")
	}

	// The first successful reservation must have been compensated (released).
	if len(svc2.releasedIDs) != 1 {
		t.Fatalf("expected 1 compensating ReleaseStock call, got %d", len(svc2.releasedIDs))
	}
	if svc2.releasedIDs[0] != svc2.firstRes.ID {
		t.Errorf("expected release of reservation %v, got %v", svc2.firstRes.ID, svc2.releasedIDs[0])
	}
}

func TestCheckoutConsumer_AllItemsFail_PublishesFailedNoCompensation(t *testing.T) {
	svc := &fakeInventoryService{reserveErr: errors.New("out of stock")}
	reservedPub := &fakePublisher{}
	failPub := &fakePublisher{}
	log := discardLogger()

	c := consumer.NewCheckoutConsumer(svc, reservedPub, failPub, log)

	evt := consumer.CheckoutRequestedEvent{
		OrderID: uuid.New().String(),
		Items:   []consumer.CheckoutItem{{SKUID: uuid.New().String(), Quantity: 5}},
	}

	if err := c.Handle(context.Background(), makeMessage(evt)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(failPub.published) != 1 {
		t.Fatalf("expected 1 failure message, got %d", len(failPub.published))
	}
	// No prior successes → nothing to compensate
	if len(svc.releasedIDs) != 0 {
		t.Fatalf("expected 0 release calls, got %d", len(svc.releasedIDs))
	}
}

// ── helper service for partial-failure test ───────────────────────────────────

type fakeInventorySvcPartial struct {
	firstSKU    uuid.UUID
	firstRes    *domain.Reservation
	failOn      uuid.UUID
	failErr     error
	releasedIDs []uuid.UUID
}

func (f *fakeInventorySvcPartial) ReserveStock(ctx context.Context, skuID, orderID uuid.UUID, qty int) (*domain.Reservation, error) {
	if skuID == f.failOn {
		return nil, f.failErr
	}
	if skuID == f.firstSKU {
		return f.firstRes, nil
	}
	return &domain.Reservation{ID: uuid.New(), SKUID: skuID}, nil
}

func (f *fakeInventorySvcPartial) ReleaseStock(ctx context.Context, reservationID uuid.UUID) error {
	f.releasedIDs = append(f.releasedIDs, reservationID)
	return nil
}
