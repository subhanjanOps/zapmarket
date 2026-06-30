package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/zapmarket/zapmarket/services/logistics-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/logistics-service/internal/infrastructure/repository"
)

type reverseCarrier interface {
	CreateReversePickup(ctx context.Context, parentAWB, orderID string) (awb, carrier, trackingURL string, err error)
}

type Handler struct {
	agents    *repository.AgentRepository
	shipments *repository.ShipmentRepository
	pods      *repository.PODRepository
	tracking  *repository.TrackingRepository
	carrier   reverseCarrier
}

func NewHandler(agents *repository.AgentRepository, shipments *repository.ShipmentRepository, pods *repository.PODRepository, tracking *repository.TrackingRepository, carrier reverseCarrier) *Handler {
	return &Handler{agents: agents, shipments: shipments, pods: pods, tracking: tracking, carrier: carrier}
}

func json200(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func json201(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// POST /v1/agents — admin only (auth checked by middleware, not here)
func (h *Handler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	var a domain.DeliveryAgent
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if a.Name == "" || a.Phone == "" || a.Zone == "" {
		jsonErr(w, http.StatusBadRequest, "name, phone, and zone are required")
		return
	}
	if err := h.agents.Create(r.Context(), &a); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to create agent")
		return
	}
	json201(w, a)
}

// GET /v1/agents
func (h *Handler) ListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := h.agents.List(r.Context())
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to list agents")
		return
	}
	json200(w, agents)
}

// PUT /v1/shipments/{id}/assign
func (h *Handler) AssignAgent(w http.ResponseWriter, r *http.Request) {
	shipmentID := r.PathValue("id")
	var body struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AgentID == "" {
		jsonErr(w, http.StatusBadRequest, "agent_id required")
		return
	}
	if err := h.shipments.AssignAgent(r.Context(), shipmentID, body.AgentID); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to assign agent")
		return
	}
	json200(w, map[string]string{"status": "assigned"})
}

// POST /v1/shipments/{id}/attempt
func (h *Handler) RecordAttempt(w http.ResponseWriter, r *http.Request) {
	shipmentID := r.PathValue("id")
	var body struct {
		Success bool `json:"success"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	shipment, err := h.shipments.GetByID(r.Context(), shipmentID)
	if err != nil {
		jsonErr(w, http.StatusNotFound, "shipment not found")
		return
	}

	undelivered, err := h.shipments.RecordAttempt(r.Context(), shipmentID, body.Success)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to record attempt")
		return
	}

	if undelivered {
		_ = h.shipments.WriteUndeliveredEvent(r.Context(), shipmentID, shipment.OrderID)
	}

	json200(w, map[string]string{"status": "recorded"})
}

// POST /v1/shipments/{id}/deliver — writes POD + marks delivered + emits outbox event atomically
func (h *Handler) Deliver(w http.ResponseWriter, r *http.Request) {
	shipmentID := r.PathValue("id")

	var body struct {
		AgentID      string  `json:"agent_id"`
		Method       string  `json:"method"`
		OTPVerified  bool    `json:"otp_verified"`
		PhotoURL     *string `json:"photo_url,omitempty"`
		COD          bool    `json:"cod"`
		AmountPaise  int64   `json:"amount_paise,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.AgentID == "" || body.Method == "" {
		jsonErr(w, http.StatusBadRequest, "agent_id and method required")
		return
	}

	shipment, err := h.shipments.GetByID(r.Context(), shipmentID)
	if err != nil {
		if err == sql.ErrNoRows {
			jsonErr(w, http.StatusNotFound, "shipment not found")
		} else {
			jsonErr(w, http.StatusInternalServerError, "failed to get shipment")
		}
		return
	}

	tx, err := h.shipments.BeginTx(r.Context())
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	pod := &domain.ProofOfDelivery{
		ShipmentID:  shipmentID,
		Method:      body.Method,
		OTPVerified: body.OTPVerified,
		PhotoURL:    body.PhotoURL,
		DeliveredBy: body.AgentID,
	}
	if err := h.pods.CreateTx(r.Context(), tx, pod); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to save proof of delivery")
		return
	}

	if _, err := tx.ExecContext(r.Context(),
		`UPDATE shipments SET status = 'DELIVERED', updated_at = NOW() WHERE id = $1`, shipmentID,
	); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to mark delivered")
		return
	}

	if err := h.shipments.WriteDeliveredEvent(r.Context(), tx, shipmentID, shipment.OrderID); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to write outbox event")
		return
	}

	if body.COD && body.AmountPaise > 0 {
		if err := h.shipments.WriteCODReconciliationTx(r.Context(), tx, shipmentID, body.AgentID, body.AmountPaise); err != nil {
			jsonErr(w, http.StatusInternalServerError, "failed to write COD reconciliation")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to commit")
		return
	}

	json200(w, map[string]any{"pod_id": pod.ID, "status": "DELIVERED"})
}

// POST /v1/webhooks/shiprocket — receives Shiprocket push tracking updates
func (h *Handler) ShiprocketWebhook(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AWB         string `json:"awb"`
		CurrentStatus string `json:"current_status"`
		Location    string `json:"current_location"`
		ShipmentID  string `json:"shipment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if payload.ShipmentID == "" {
		json200(w, map[string]string{"status": "ok"})
		return
	}
	if err := h.tracking.Insert(r.Context(), payload.ShipmentID, payload.CurrentStatus, "", payload.Location); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to record tracking event")
		return
	}
	json200(w, map[string]string{"status": "ok"})
}

// GET /v1/shipments/{id}/tracking — returns tracking events for a shipment
func (h *Handler) GetTracking(w http.ResponseWriter, r *http.Request) {
	shipmentID := r.PathValue("id")
	events, err := h.tracking.List(r.Context(), shipmentID)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to list tracking events")
		return
	}
	json200(w, events)
}

// POST /v1/return-shipments — creates a reverse pickup for a parent shipment
func (h *Handler) CreateReturnShipment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ParentShipmentID string `json:"parent_shipment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ParentShipmentID == "" {
		jsonErr(w, http.StatusBadRequest, "parent_shipment_id required")
		return
	}

	parent, err := h.shipments.GetByID(r.Context(), body.ParentShipmentID)
	if err != nil {
		if err == sql.ErrNoRows {
			jsonErr(w, http.StatusNotFound, "parent shipment not found")
		} else {
			jsonErr(w, http.StatusInternalServerError, "failed to get parent shipment")
		}
		return
	}

	awb, carrier, trackingURL, err := h.carrier.CreateReversePickup(r.Context(), parent.CarrierShipmentID, parent.OrderID)
	if err != nil {
		jsonErr(w, http.StatusBadGateway, "carrier error: "+err.Error())
		return
	}

	reverse, err := h.shipments.CreateReverse(r.Context(), parent.ID, parent.OrderID, awb, carrier, trackingURL)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to save reverse shipment")
		return
	}

	json201(w, reverse)
}
