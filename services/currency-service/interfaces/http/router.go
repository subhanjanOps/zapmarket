package http

import (
	"fmt"
	"net/http"

	"github.com/zapmarket/zapmarket/services/currency-service/interfaces/metrics"
)

// NewRouter builds the HTTP mux for the currency service.
func NewRouter(h *Handler, m *metrics.Metrics) http.Handler {
	mux := http.NewServeMux()

	// Public endpoints — no auth (gateway enforces auth_mode=none for these paths)
	mux.HandleFunc("GET /v1/currencies", h.ListCurrencies)
	mux.HandleFunc("GET /v1/currencies/rates", h.GetRates)

	// Admin endpoint — gateway enforces JWT role=admin before forwarding here
	mux.HandleFunc("PUT /v1/admin/currencies/{code}", h.ToggleCurrency)

	// Prometheus metrics
	mux.Handle("/metrics", m.Handler())

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok","service":"currency-service"}`)
	})

	return mux
}
