package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type EventHandler struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewEventHandler(db *sql.DB, logger *slog.Logger) *EventHandler {
	return &EventHandler{db: db, logger: logger}
}

type eventRequest struct {
	UserID     *string         `json:"user_id"`
	SessionID  string          `json:"session_id"`
	EventType  string          `json:"event_type"`
	ProductID  *string         `json:"product_id"`
	CategoryID *string         `json:"category_id"`
	Metadata   json.RawMessage `json:"metadata"`
}

// RecordEvent handles POST /v1/events — fire-and-forget, always returns 202.
func (h *EventHandler) RecordEvent(w http.ResponseWriter, r *http.Request) {
	var req eventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusAccepted) // best-effort; don't block the client
		return
	}

	go func() {
		var userID, productID, categoryID *uuid.UUID
		if req.UserID != nil {
			if id, err := uuid.Parse(*req.UserID); err == nil {
				userID = &id
			}
		}
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
		_, err := h.db.Exec(`
			INSERT INTO user_events (user_id, session_id, event_type, product_id, category_id, metadata, occurred_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, userID, req.SessionID, req.EventType, productID, categoryID, meta, time.Now())
		if err != nil {
			h.logger.Error("failed to insert user event", "error", err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}
