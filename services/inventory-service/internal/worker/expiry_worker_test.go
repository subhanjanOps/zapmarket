package worker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/inventory-service/internal/worker"
)

// fakeRepo is a thread-safe fake for reservationReleaser.
type fakeRepo struct {
	mu       sync.Mutex
	expired  []*domain.Reservation
	released []uuid.UUID
}

func (f *fakeRepo) FindExpiredReservations(_ context.Context, _ time.Time) ([]*domain.Reservation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.expired, nil
}

func (f *fakeRepo) ReleaseStock(_ context.Context, id uuid.UUID) (uuid.UUID, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.released = append(f.released, id)
	return uuid.New(), 1, nil
}

func (f *fakeRepo) WriteReservationExpiredEvent(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (f *fakeRepo) getReleased() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]uuid.UUID, len(f.released))
	copy(out, f.released)
	return out
}

func TestExpiryWorker_ReleasesExpiredReservations(t *testing.T) {
	res1ID := uuid.New()
	res2ID := uuid.New()

	repo := &fakeRepo{
		expired: []*domain.Reservation{
			{ID: res1ID, OrderID: uuid.New(), SKUID: uuid.New(), Qty: 2, Status: domain.ReservationReserved, ExpiresAt: time.Now().Add(-1 * time.Minute)},
			{ID: res2ID, OrderID: uuid.New(), SKUID: uuid.New(), Qty: 1, Status: domain.ReservationReserved, ExpiresAt: time.Now().Add(-2 * time.Minute)},
		},
	}

	w := worker.NewExpiryWorker(repo, 50*time.Millisecond, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	// Wait for at least one tick to fire.
	time.Sleep(120 * time.Millisecond)

	released := repo.getReleased()
	if len(released) < 2 {
		t.Fatalf("expected at least 2 releases after first tick, got %d", len(released))
	}

	// Verify both reservation IDs were released.
	releaseSet := make(map[uuid.UUID]bool)
	for _, id := range released {
		releaseSet[id] = true
	}
	if !releaseSet[res1ID] {
		t.Errorf("res1 (%v) was not released", res1ID)
	}
	if !releaseSet[res2ID] {
		t.Errorf("res2 (%v) was not released", res2ID)
	}

	cancel()
	<-done
}

func TestExpiryWorker_SkipsNonExpiredReservations(t *testing.T) {
	repo := &fakeRepo{
		// FindExpiredReservations returns empty — no expired rows.
		expired: []*domain.Reservation{},
	}

	w := worker.NewExpiryWorker(repo, 50*time.Millisecond, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	time.Sleep(120 * time.Millisecond)

	if len(repo.getReleased()) != 0 {
		t.Fatalf("expected 0 releases, got %d", len(repo.getReleased()))
	}

	cancel()
	<-done
}
