package usecases_test

import (
	"context"
	"testing"
	"time"

	"github.com/zapmarket/zapmarket/services/promotions-service/internal/application/usecases"
	"github.com/zapmarket/zapmarket/services/promotions-service/internal/domain"
	domainerrors "github.com/zapmarket/zapmarket/services/promotions-service/internal/domain/errors"
)

type fakeCouponRepo struct{ coupon *domain.Coupon }

func (f *fakeCouponRepo) FindByCode(_ context.Context, code string) (*domain.Coupon, error) {
	if f.coupon != nil && f.coupon.Code == code {
		return f.coupon, nil
	}
	return nil, domainerrors.ErrCouponNotFound
}

func (f *fakeCouponRepo) CountUsageByUser(_ context.Context, _, _ string) (int, error) { return 0, nil }
func (f *fakeCouponRepo) CountUsageTotal(_ context.Context, _ string) (int, error)      { return 0, nil }

func TestValidateCoupon_AppliesPercentDiscount(t *testing.T) {
	expires := time.Now().Add(24 * time.Hour)
	repo := &fakeCouponRepo{coupon: &domain.Coupon{
		Code:           "SAVE10",
		DiscountType:   domain.DiscountTypePercent,
		DiscountValue:  10,
		MinOrderPaise:  50000,
		MaxUsesPerUser: 1,
		ExpiresAt:      &expires,
		IsActive:       true,
	}}
	uc := usecases.NewValidateCouponUseCase(repo)

	result, err := uc.Execute(context.Background(), usecases.ValidateCouponInput{
		Code:           "SAVE10",
		UserID:         "u1",
		CartTotalPaise: 100000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DiscountPaise != 10000 {
		t.Fatalf("expected discount 10000, got %d", result.DiscountPaise)
	}
}

func TestValidateCoupon_RejectsExpiredCoupon(t *testing.T) {
	expired := time.Now().Add(-1 * time.Hour)
	repo := &fakeCouponRepo{coupon: &domain.Coupon{
		Code:      "OLD10",
		IsActive:  true,
		ExpiresAt: &expired,
	}}
	uc := usecases.NewValidateCouponUseCase(repo)

	_, err := uc.Execute(context.Background(), usecases.ValidateCouponInput{
		Code: "OLD10", UserID: "u1", CartTotalPaise: 100000,
	})
	if err == nil {
		t.Fatal("expected error for expired coupon")
	}
}
