package http

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/zapmarket/zapmarket/services/review-return-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/review-return-service/internal/infrastructure/postgres"
)

type Handler struct {
	repo              *postgres.Repository
	logisticsBaseURL  string
}

func New(repo *postgres.Repository) *Handler {
	u := os.Getenv("LOGISTICS_SERVICE_URL")
	if u == "" {
		u = "http://logistics-service:8092"
	}
	return &Handler{repo: repo, logisticsBaseURL: u}
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// POST /v1/returns
func (h *Handler) CreateReturn(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OrderID     string   `json:"order_id"`
		OrderItemID string   `json:"order_item_id"`
		Reason      string   `json:"reason"`
		Description string   `json:"description"`
		ImageURLs   []string `json:"image_urls"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	req := &domain.ReturnRequest{
		OrderID:     body.OrderID,
		OrderItemID: body.OrderItemID,
		Reason:      body.Reason,
		Description: body.Description,
		ImageURLs:   body.ImageURLs,
	}
	if err := h.repo.CreateReturn(r.Context(), req); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to create return")
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, req)
}

// PUT /v1/returns/{id}/approve
func (h *Handler) ApproveReturn(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ret, err := h.repo.GetReturnByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonErr(w, http.StatusNotFound, "return not found")
		} else {
			jsonErr(w, http.StatusInternalServerError, "failed to get return")
		}
		return
	}
	if ret.Status != "REQUESTED" {
		jsonErr(w, http.StatusConflict, "return is not in REQUESTED state")
		return
	}

	// Create reverse shipment in logistics-service
	reverseShipmentID, err := h.createReverseShipment(r.Context(), id)
	if err != nil {
		jsonErr(w, http.StatusBadGateway, "failed to create reverse shipment: "+err.Error())
		return
	}

	if err := h.repo.ApproveReturn(r.Context(), id, reverseShipmentID); err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to approve return")
		return
	}
	jsonOK(w, map[string]string{"status": "APPROVED", "reverse_shipment_id": reverseShipmentID})
}

// PUT /v1/returns/{id}/reject
func (h *Handler) RejectReturn(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if err := h.repo.RejectReturn(r.Context(), id, body.Reason); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonErr(w, http.StatusNotFound, "return not found")
		} else {
			jsonErr(w, http.StatusInternalServerError, "failed to reject return")
		}
		return
	}
	jsonOK(w, map[string]string{"status": "REJECTED"})
}

// GET /v1/products/{id}/rating
func (h *Handler) GetProductRating(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	rating, err := h.repo.GetProductRating(r.Context(), productID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonOK(w, &domain.ProductRating{ProductID: productID, AvgRating: 0, ReviewCount: 0})
		} else {
			jsonErr(w, http.StatusInternalServerError, "failed to get rating")
		}
		return
	}
	jsonOK(w, rating)
}

func (h *Handler) createReverseShipment(ctx context.Context, returnID string) (string, error) {
	payload, _ := json.Marshal(map[string]string{"parent_shipment_id": returnID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.logisticsBaseURL+"/v1/return-shipments", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("logistics-service HTTP %d: %s", resp.StatusCode, body)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.ID, nil
}
