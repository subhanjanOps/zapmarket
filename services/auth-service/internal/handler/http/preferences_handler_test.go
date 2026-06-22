package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	handler "github.com/zapmarket/zapmarket/services/auth-service/internal/handler/http"
)

type fakePrefsRepo struct {
	stored map[string]string
}

func (f *fakePrefsRepo) Get(_ context.Context, _ uuid.UUID, key string) (string, bool, error) {
	v, ok := f.stored[key]
	return v, ok, nil
}

func (f *fakePrefsRepo) Set(_ context.Context, _ uuid.UUID, key, value string) error {
	f.stored[key] = value
	return nil
}

func TestGetPreferences_ReturnsStoredCurrency(t *testing.T) {
	repo := &fakePrefsRepo{stored: map[string]string{"display_currency": "EUR"}}
	h := handler.NewPreferencesHandler(repo, nil)

	userID := uuid.New()
	ctx := context.WithValue(context.Background(), handler.PrefsUserIDKey, userID)
	req := httptest.NewRequest(http.MethodGet, "/v1/users/me/preferences", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetPreferences(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["display_currency"] != "EUR" {
		t.Errorf("expected EUR, got %q", resp["display_currency"])
	}
}

func TestSetPreferences_RejectsBadCurrencyCode(t *testing.T) {
	repo := &fakePrefsRepo{stored: map[string]string{}}
	h := handler.NewPreferencesHandler(repo, nil)

	userID := uuid.New()
	ctx := context.WithValue(context.Background(), handler.PrefsUserIDKey, userID)
	body := bytes.NewBufferString(`{"display_currency":"usd"}`) // lowercase — invalid
	req := httptest.NewRequest(http.MethodPut, "/v1/users/me/preferences", body).WithContext(ctx)
	w := httptest.NewRecorder()

	h.SetPreferences(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
