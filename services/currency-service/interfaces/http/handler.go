package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/zapmarket/zapmarket/services/currency-service/application/usecases"
)

// Handler holds all HTTP handlers for the currency service.
type Handler struct {
	listCurrencies  *usecases.ListCurrenciesUseCase
	getRates        *usecases.GetRatesUseCase
	toggleCurrency  *usecases.ToggleCurrencyUseCase
	getRatesHistory *usecases.GetRatesHistoryUseCase
	log             *slog.Logger
}

func NewHandler(
	listCurrencies *usecases.ListCurrenciesUseCase,
	getRates *usecases.GetRatesUseCase,
	toggleCurrency *usecases.ToggleCurrencyUseCase,
	getRatesHistory *usecases.GetRatesHistoryUseCase,
	log *slog.Logger,
) *Handler {
	return &Handler{
		listCurrencies:  listCurrencies,
		getRates:        getRates,
		toggleCurrency:  toggleCurrency,
		getRatesHistory: getRatesHistory,
		log:             log,
	}
}

// ListCurrencies handles GET /v1/currencies — public, no auth.
func (h *Handler) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	currencies, err := h.listCurrencies.Execute(r.Context())
	if err != nil {
		h.log.Error("list currencies", "error", err)
		h.writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.writeJSON(w, http.StatusOK, currencies)
}

// GetRates handles GET /v1/currencies/rates — public, no auth.
func (h *Handler) GetRates(w http.ResponseWriter, r *http.Request) {
	ratesDTO, err := h.getRates.Execute(r.Context(), "USD")
	if err != nil {
		var staleErr *usecases.ErrRatesTooStale
		if errors.As(err, &staleErr) {
			h.writeError(w, http.StatusServiceUnavailable, "exchange rates unavailable: "+staleErr.Error())
			return
		}
		h.log.Error("get rates", "error", err)
		h.writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	h.writeJSON(w, http.StatusOK, ratesDTO)
}

// ToggleCurrency handles PUT /v1/admin/currencies/{code}.
// The gateway enforces JWT auth (auth_mode=required); this handler additionally
// verifies that the X-User-Role header (set by gateway after token validation)
// is "admin".
func (h *Handler) ToggleCurrency(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-User-Role") != "admin" {
		h.writeError(w, http.StatusForbidden, "admin role required")
		return
	}
	code := r.PathValue("code")
	if code == "" {
		h.writeError(w, http.StatusBadRequest, "currency code is required")
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.toggleCurrency.Execute(r.Context(), code, body.Enabled); err != nil {
		h.log.Error("toggle currency", "code", code, "enabled", body.Enabled, "error", err)
		h.writeError(w, http.StatusNotFound, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"code":    code,
		"enabled": body.Enabled,
	})
}

// GetRatesHistory handles GET /v1/currencies/rates/history?date=YYYY-MM-DD
func (h *Handler) GetRatesHistory(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		h.writeError(w, http.StatusBadRequest, "date query parameter is required (YYYY-MM-DD)")
		return
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "date must be in YYYY-MM-DD format")
		return
	}
	dto, err := h.getRatesHistory.Execute(r.Context(), "USD", date)
	if err != nil {
		h.log.Error("get rates history", "error", err)
		h.writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	h.writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, map[string]string{"error": msg})
}
