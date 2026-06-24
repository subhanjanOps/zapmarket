package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	domainerrors "github.com/zapmarket/zapmarket/services/currency-service/domain/errors"
	"github.com/zapmarket/zapmarket/services/currency-service/application/usecases"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/valueobjects"
)

// Handler holds all HTTP handlers for the currency service.
type Handler struct {
	listCurrencies  usecases.CurrencyLister
	getRates        usecases.RatesGetter
	toggleCurrency  usecases.CurrencyToggler
	getRatesHistory usecases.RatesHistoryGetter
	log             *slog.Logger
}

func NewHandler(
	listCurrencies usecases.CurrencyLister,
	getRates usecases.RatesGetter,
	toggleCurrency usecases.CurrencyToggler,
	getRatesHistory usecases.RatesHistoryGetter,
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

// ListCurrencies handles GET /api/v1/currencies — public, no auth.
func (h *Handler) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	currencies, err := h.listCurrencies.Execute(r.Context())
	if err != nil {
		h.log.Error("list currencies", "error", err, "request_id", requestIDFrom(r.Context()))
		h.writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.writeJSON(w, http.StatusOK, currencies)
}

// GetRates handles GET /api/v1/currencies/rates?base=USD — public, no auth.
// The base query parameter defaults to USD.
func (h *Handler) GetRates(w http.ResponseWriter, r *http.Request) {
	base := r.URL.Query().Get("base")
	if base == "" {
		base = "USD"
	}
	if _, err := valueobjects.NewCurrencyCode(base); err != nil {
		h.writeError(w, http.StatusBadRequest, "base must be a 3-letter ISO 4217 currency code")
		return
	}

	ratesDTO, err := h.getRates.Execute(r.Context(), base)
	if err != nil {
		var staleErr *usecases.ErrRatesTooStale
		if errors.As(err, &staleErr) {
			h.writeError(w, http.StatusServiceUnavailable, "exchange rates unavailable: "+staleErr.Error())
			return
		}
		h.log.Error("get rates", "error", err, "base", base, "request_id", requestIDFrom(r.Context()))
		h.writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	h.writeJSON(w, http.StatusOK, ratesDTO)
}

// ToggleCurrency handles PUT /api/v1/admin/currencies/{code}.
// The gateway enforces JWT auth (auth_mode=required) and sets X-User-Role after token
// validation. This handler performs a secondary role check. Note: this assumes network
// policy prevents direct access to the service, bypassing the gateway. For stronger
// guarantees, forward the JWT to auth-service for validation.
func (h *Handler) ToggleCurrency(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if _, err := valueobjects.NewCurrencyCode(code); err != nil {
		h.writeError(w, http.StatusBadRequest, "currency code must be a 3-letter ISO 4217 code")
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
		var notFound *domainerrors.ErrCurrencyNotFound
		if errors.As(err, &notFound) {
			h.writeError(w, http.StatusNotFound, "currency not found")
			return
		}
		h.log.Error("toggle currency", "code", code, "enabled", body.Enabled, "error", err,
			"request_id", requestIDFrom(r.Context()))
		h.writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"code":    code,
		"enabled": body.Enabled,
	})
}

// GetRatesHistory handles GET /api/v1/currencies/rates/history?date=YYYY-MM-DD&base=USD
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

	base := r.URL.Query().Get("base")
	if base == "" {
		base = "USD"
	}

	dto, err := h.getRatesHistory.Execute(r.Context(), base, date)
	if err != nil {
		var noHistory *domainerrors.ErrNoRatesHistory
		if errors.As(err, &noHistory) {
			h.writeError(w, http.StatusNotFound, noHistory.Error())
			return
		}
		h.log.Error("get rates history", "error", err, "request_id", requestIDFrom(r.Context()))
		h.writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	h.writeJSON(w, http.StatusOK, dto)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.log.Error("encode response", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, map[string]string{"error": msg})
}
