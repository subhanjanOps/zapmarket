package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/review-return-service/internal/domain"
)

var ErrNotVerifiedPurchase = errors.New("review requires a verified purchase for this SKU")

type orderVerifier interface {
	IsOrderConfirmedForSKU(ctx context.Context, userID, orderID, skuID string) (bool, error)
}

type reviewSaver interface {
	Save(ctx context.Context, review *domain.Review) error
}

type SubmitReviewInput struct {
	UserID    string
	OrderID   string
	ProductID string
	SKUID     string
	Rating    int
	Title     string
	Body      string
	ImageURLs []string
}

type SubmitReviewUseCase struct {
	repo     reviewSaver
	verifier orderVerifier
}

func NewSubmitReviewUseCase(repo reviewSaver, verifier orderVerifier) *SubmitReviewUseCase {
	return &SubmitReviewUseCase{repo: repo, verifier: verifier}
}

func (uc *SubmitReviewUseCase) Execute(ctx context.Context, in SubmitReviewInput) error {
	confirmed, err := uc.verifier.IsOrderConfirmedForSKU(ctx, in.UserID, in.OrderID, in.SKUID)
	if err != nil {
		return err
	}
	if !confirmed {
		return ErrNotVerifiedPurchase
	}
	review := &domain.Review{
		ID:               uuid.NewString(),
		ProductID:        in.ProductID,
		SKUID:            in.SKUID,
		UserID:           in.UserID,
		OrderID:          in.OrderID,
		VerifiedPurchase: true,
		Rating:           in.Rating,
		Title:            in.Title,
		Body:             in.Body,
		ImageURLs:        in.ImageURLs,
		Status:           "PENDING_MODERATION",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	return uc.repo.Save(ctx, review)
}
