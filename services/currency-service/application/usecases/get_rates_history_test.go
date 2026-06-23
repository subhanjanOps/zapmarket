package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domainerrors "github.com/zapmarket/zapmarket/services/currency-service/domain/errors"

	"github.com/zapmarket/zapmarket/services/currency-service/application/usecases"
	"github.com/zapmarket/zapmarket/services/currency-service/domain/entities"
)

type fakeHistoryRepo struct {
	rows []entities.ExchangeRate
}

func (f *fakeHistoryRepo) HistoryByBase(_ context.Context, base string, date time.Time) ([]entities.ExchangeRate, error) {
	return f.rows, nil
}

func TestGetRatesHistory_ReturnsDTO(t *testing.T) {
	date := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := &fakeHistoryRepo{rows: []entities.ExchangeRate{
		{Base: "USD", Quote: "EUR", Rate: 0.92, AsOf: date},
		{Base: "USD", Quote: "GBP", Rate: 0.79, AsOf: date},
	}}

	uc := usecases.NewGetRatesHistoryUseCase(repo)
	dto, err := uc.Execute(context.Background(), "USD", date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dto.Base != "USD" {
		t.Errorf("expected base USD, got %s", dto.Base)
	}
	if dto.Rates["EUR"] != 0.92 {
		t.Errorf("expected EUR=0.92, got %v", dto.Rates["EUR"])
	}
	if len(dto.Rates) != 2 {
		t.Errorf("expected 2 rates, got %d", len(dto.Rates))
	}
}

func TestGetRatesHistory_ReturnsErrNoRatesHistoryOnNoData(t *testing.T) {
	repo := &fakeHistoryRepo{rows: nil}
	uc := usecases.NewGetRatesHistoryUseCase(repo)
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := uc.Execute(context.Background(), "USD", date)
	if err == nil {
		t.Fatal("expected ErrNoRatesHistory, got nil")
	}
	var noHistory *domainerrors.ErrNoRatesHistory
	if !errors.As(err, &noHistory) {
		t.Errorf("expected ErrNoRatesHistory, got %T: %v", err, err)
	}
}
