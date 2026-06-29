package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/application"
)

type BalanceHandler struct{ ledger application.LedgerRepository }

func NewBalanceHandler(ledger application.LedgerRepository) *BalanceHandler {
	return &BalanceHandler{ledger: ledger}
}

func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	sellerID := chi.URLParam(r, "id")
	bal, err := h.ledger.GetBalance(r.Context(), sellerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bal)
}
