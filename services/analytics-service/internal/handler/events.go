package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/pkg/crypto"
)

// eventRow is a validated, ready-to-insert event.
type eventRow struct {
	UserID     *uuid.UUID
	SessionID  string
	EventType  string
	ProductID  *uuid.UUID
	CategoryID *uuid.UUID
	Metadata   []byte
	OccurredAt time.Time
}

// EventWriter owns a bounded queue and a fixed pool of workers that persist
// events to Postgres. Unlike a bare `go func(){ ... }()` per request, it has
// a bounded backlog (so a slow DB applies backpressure instead of unbounded
// goroutine growth) and a Close method that drains in-flight work on
// shutdown instead of dropping it.
type EventWriter struct {
	db     *sql.DB
	logger *slog.Logger
	queue  chan eventRow
	done   chan struct{}
}

func NewEventWriter(db *sql.DB, logger *slog.Logger, queueSize, workers int) *EventWriter {
	w := &EventWriter{
		db:     db,
		logger: logger,
		queue:  make(chan eventRow, queueSize),
		done:   make(chan struct{}),
	}
	go w.run(workers)
	return w
}

func (w *EventWriter) run(workers int) {
	workerDone := make(chan struct{}, workers)
	for i := 0; i < workers; i++ {
		go func() {
			for row := range w.queue {
				w.insert(row)
			}
			workerDone <- struct{}{}
		}()
	}
	for i := 0; i < workers; i++ {
		<-workerDone
	}
	close(w.done)
}

func (w *EventWriter) insert(row eventRow) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := w.db.ExecContext(ctx, `
		INSERT INTO user_events (user_id, session_id, event_type, product_id, category_id, metadata, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, row.UserID, row.SessionID, row.EventType, row.ProductID, row.CategoryID, row.Metadata, row.OccurredAt)
	if err != nil {
		w.logger.Error("failed to insert user event", "error", err)
	}
}

// Enqueue submits an event for asynchronous persistence. Returns false if the
// queue is full (caller should still respond 202; the event is dropped and
// counted, rather than spawning an unbounded goroutine).
func (w *EventWriter) Enqueue(row eventRow) bool {
	select {
	case w.queue <- row:
		return true
	default:
		w.logger.Warn("event queue full, dropping event", "event_type", row.EventType)
		return false
	}
}

// Close stops accepting new events and waits for queued events to drain.
func (w *EventWriter) Close(ctx context.Context) {
	close(w.queue)
	select {
	case <-w.done:
	case <-ctx.Done():
	}
}

type EventHandler struct {
	writer *EventWriter
	logger *slog.Logger
}

func NewEventHandler(writer *EventWriter, logger *slog.Logger) *EventHandler {
	return &EventHandler{writer: writer, logger: logger}
}

type eventRequest struct {
	// UserID is only honored for unauthenticated (anonymous) requests are
	// rejected identity — an authenticated caller's user_id always comes from
	// their validated token, never from this field, to prevent spoofing.
	UserID     *string         `json:"user_id"`
	SessionID  string          `json:"session_id"`
	EventType  string          `json:"event_type"`
	ProductID  *string         `json:"product_id"`
	CategoryID *string         `json:"category_id"`
	Metadata   json.RawMessage `json:"metadata"`
}

// RecordEvent handles POST /v1/events — fire-and-forget, always returns 202.
// Authentication is optional (anonymous browsing events are valid), but when
// a Bearer token is present it must be valid, and the resulting user_id
// always overrides any client-supplied value.
func (h *EventHandler) RecordEvent(w http.ResponseWriter, r *http.Request) {
	var req eventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.SessionID == "" || req.EventType == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var userID *uuid.UUID
	if claims, ok := crypto.ClaimsFromContext(r.Context()); ok {
		userID = &claims.UserID
	}
	// Anonymous requests never get to set user_id themselves.

	var productID, categoryID *uuid.UUID
	if req.ProductID != nil {
		if id, err := uuid.Parse(*req.ProductID); err == nil {
			productID = &id
		}
	}
	if req.CategoryID != nil {
		if id, err := uuid.Parse(*req.CategoryID); err == nil {
			categoryID = &id
		}
	}
	meta := []byte("{}")
	if len(req.Metadata) > 0 {
		meta = req.Metadata
	}

	h.writer.Enqueue(eventRow{
		UserID:     userID,
		SessionID:  req.SessionID,
		EventType:  req.EventType,
		ProductID:  productID,
		CategoryID: categoryID,
		Metadata:   meta,
		OccurredAt: time.Now(),
	})

	w.WriteHeader(http.StatusAccepted)
}
