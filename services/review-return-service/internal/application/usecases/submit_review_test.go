package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zapmarket/zapmarket/services/review-return-service/internal/application/usecases"
	"github.com/zapmarket/zapmarket/services/review-return-service/internal/domain"
)

type fakeOrderVerifier struct{ confirmed bool }

func (f *fakeOrderVerifier) IsOrderConfirmedForSKU(_ context.Context, _, _, _ string) (bool, error) {
	return f.confirmed, nil
}

type fakeReviewRepo struct{ saved bool }

func (f *fakeReviewRepo) Save(_ context.Context, _ *domain.Review) error {
	f.saved = true
	return nil
}

func TestSubmitReview_RequiresVerifiedPurchase(t *testing.T) {
	verifier := &fakeOrderVerifier{confirmed: false}
	repo := &fakeReviewRepo{}
	uc := usecases.NewSubmitReviewUseCase(repo, verifier)

	err := uc.Execute(context.Background(), usecases.SubmitReviewInput{
		UserID: "u1", OrderID: "o1", SKUID: "s1", Rating: 5, Title: "Great",
	})
	if !errors.Is(err, usecases.ErrNotVerifiedPurchase) {
		t.Fatalf("expected ErrNotVerifiedPurchase, got %v", err)
	}
	if repo.saved {
		t.Fatal("review should not be saved without verified purchase")
	}
}

func TestSubmitReview_SavesWhenVerified(t *testing.T) {
	verifier := &fakeOrderVerifier{confirmed: true}
	repo := &fakeReviewRepo{}
	uc := usecases.NewSubmitReviewUseCase(repo, verifier)

	err := uc.Execute(context.Background(), usecases.SubmitReviewInput{
		UserID: "u1", OrderID: "o1", SKUID: "s1", Rating: 4, Title: "Good product",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.saved {
		t.Fatal("expected review to be saved")
	}
}
