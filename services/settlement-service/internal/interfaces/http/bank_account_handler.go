package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/domain"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/infrastructure/postgres"
)

type bankAccountStore interface {
	Create(ctx context.Context, a *domain.SellerBankAccount) error
	ListBySeller(ctx context.Context, sellerID string) ([]*domain.SellerBankAccount, error)
}

type BankAccountHandler struct{ repo bankAccountStore }

func NewBankAccountHandler(repo *postgres.BankAccountRepo) *BankAccountHandler {
	return &BankAccountHandler{repo: repo}
}

func (h *BankAccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	sellerID := chi.URLParam(r, "id")
	var a domain.SellerBankAccount
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	a.SellerID = sellerID
	if a.AccountNumber == "" || a.IFSCCode == "" || a.AccountHolderName == "" {
		http.Error(w, `{"error":"account_holder_name, account_number, ifsc_code required"}`, http.StatusBadRequest)
		return
	}
	if err := h.repo.Create(r.Context(), &a); err != nil {
		http.Error(w, `{"error":"failed to save bank account"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

func (h *BankAccountHandler) List(w http.ResponseWriter, r *http.Request) {
	sellerID := chi.URLParam(r, "id")
	accounts, err := h.repo.ListBySeller(r.Context(), sellerID)
	if err != nil {
		http.Error(w, `{"error":"failed to list accounts"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}
