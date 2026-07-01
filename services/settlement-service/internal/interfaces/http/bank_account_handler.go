package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zapmarket/zapmarket/pkg/crypto"
	"github.com/zapmarket/zapmarket/services/settlement-service/internal/domain"
)

type bankAccountStore interface {
	Create(ctx context.Context, a *domain.SellerBankAccount) error
	ListBySeller(ctx context.Context, sellerID string) ([]*domain.SellerBankAccount, error)
}

type BankAccountHandler struct{ repo bankAccountStore }

func NewBankAccountHandler(repo bankAccountStore) *BankAccountHandler {
	return &BankAccountHandler{repo: repo}
}

// authorizeSeller ensures the authenticated caller is either the seller
// identified by the path param or an admin. Prevents any seller from
// reading or writing another seller's balance/bank details.
func authorizeSeller(r *http.Request, pathSellerID string) bool {
	claims, ok := crypto.ClaimsFromContext(r.Context())
	if !ok {
		return false
	}
	return claims.Role == "admin" || claims.UserID.String() == pathSellerID
}

func (h *BankAccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	sellerID := chi.URLParam(r, "id")
	if !authorizeSeller(r, sellerID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
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
	if !authorizeSeller(r, sellerID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	accounts, err := h.repo.ListBySeller(r.Context(), sellerID)
	if err != nil {
		http.Error(w, `{"error":"failed to list accounts"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}
