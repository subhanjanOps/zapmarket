package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zapmarket/zapmarket/services/currency-service/infrastructure/metrics"
)

// NewRouter builds the HTTP mux for the currency service.
// liveness is called by the /health endpoint to verify downstream dependencies.
// authMW must be non-nil; it validates JWTs for admin routes via auth-service gRPC.
func NewRouter(h *Handler, m *metrics.Metrics, liveness func(context.Context) error, authMW *AuthMiddleware) http.Handler {
	mux := http.NewServeMux()

	// Public endpoints — no auth (gateway enforces auth_mode=none for these paths).
	mux.HandleFunc("GET /api/v1/currencies", h.ListCurrencies)
	mux.HandleFunc("GET /api/v1/currencies/rates/history", h.GetRatesHistory)
	mux.HandleFunc("GET /api/v1/currencies/rates", h.GetRates)

	// Admin endpoint — JWT validated by auth-service; only admin role permitted.
	mux.Handle("PUT /api/v1/admin/currencies/{code}", authMW.RequireRole("admin")(http.HandlerFunc(h.ToggleCurrency)))

	// Prometheus metrics — internal only, not exposed via gateway.
	mux.Handle("/metrics", m.Handler())

	// Health / readiness — checks DB and Redis so load-balancers can drain unhealthy instances.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := liveness(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "unhealthy",
				"service": "currency-service",
				"error":   err.Error(),
			})
			return
		}
		fmt.Fprint(w, `{"status":"ok","service":"currency-service"}`)
	})

	return RequestID(mux)
}
