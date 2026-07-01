package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zapmarket/zapmarket/services/promotions-service/internal/domain"
)

var (
	ErrCouponExpired     = errors.New("coupon has expired")
	ErrCouponInactive    = errors.New("coupon is not active")
	ErrCartBelowMinimum  = errors.New("cart total is below coupon minimum order value")
	ErrUsageLimitReached = errors.New("coupon usage limit has been reached")
)

type couponRepo interface {
	FindByCode(ctx context.Context, code string) (*domain.Coupon, error)
	CountUsageByUser(ctx context.Context, couponID, userID string) (int, error)
	CountUsageTotal(ctx context.Context, couponID string) (int, error)
}

type ValidateCouponInput struct {
	Code           string
	UserID         string
	CartTotalPaise int64
}

type ValidateCouponResult struct {
	CouponID      string
	DiscountPaise int64
	FinalPaise    int64
}

type ValidateCouponUseCase struct{ repo couponRepo }

func NewValidateCouponUseCase(repo couponRepo) *ValidateCouponUseCase {
	return &ValidateCouponUseCase{repo: repo}
}

func (uc *ValidateCouponUseCase) Execute(ctx context.Context, in ValidateCouponInput) (*ValidateCouponResult, error) {
	coupon, err := uc.repo.FindByCode(ctx, strings.ToUpper(in.Code))
	if err != nil {
		return nil, err
	}
	if !coupon.IsActive {
		return nil, ErrCouponInactive
	}
	if coupon.ExpiresAt != nil && time.Now().After(*coupon.ExpiresAt) {
		return nil, ErrCouponExpired
	}
	if in.CartTotalPaise < coupon.MinOrderPaise {
		return nil, ErrCartBelowMinimum
	}
	if coupon.MaxUsesPerUser > 0 {
		used, err := uc.repo.CountUsageByUser(ctx, coupon.ID, in.UserID)
		if err != nil {
			return nil, err
		}
		if used >= coupon.MaxUsesPerUser {
			return nil, ErrUsageLimitReached
		}
	}
	if coupon.MaxUsesTotal > 0 {
		total, err := uc.repo.CountUsageTotal(ctx, coupon.ID)
		if err != nil {
			return nil, err
		}
		if total >= coupon.MaxUsesTotal {
			return nil, ErrUsageLimitReached
		}
	}
	discount := coupon.Apply(in.CartTotalPaise)
	return &ValidateCouponResult{
		CouponID:      coupon.ID,
		DiscountPaise: discount,
		FinalPaise:    in.CartTotalPaise - discount,
	}, nil
}
