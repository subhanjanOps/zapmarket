package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zapmarket/zapmarket/services/cart-service/internal/domain"
)

const guestTTL = 7 * 24 * time.Hour

type GuestCartRepo struct{ rdb *goredis.Client }

func NewGuestCartRepo(rdb *goredis.Client) *GuestCartRepo { return &GuestCartRepo{rdb: rdb} }

func guestKey(sessionID string) string { return fmt.Sprintf("cart:guest:%s", sessionID) }

func (r *GuestCartRepo) GetGuestCart(ctx context.Context, sessionID string) (*domain.Cart, error) {
	data, err := r.rdb.Get(ctx, guestKey(sessionID)).Bytes()
	if err == goredis.Nil {
		return &domain.Cart{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cart domain.Cart
	if err := json.Unmarshal(data, &cart); err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *GuestCartRepo) ClearGuestCart(ctx context.Context, sessionID string) error {
	return r.rdb.Del(ctx, guestKey(sessionID)).Err()
}

func (r *GuestCartRepo) UpsertItem(ctx context.Context, sessionID string, item domain.CartItem) error {
	cart, err := r.GetGuestCart(ctx, sessionID)
	if err != nil {
		return err
	}
	found := false
	for i, existing := range cart.Items {
		if existing.SKUID == item.SKUID {
			cart.Items[i] = item
			found = true
			break
		}
	}
	if !found {
		cart.Items = append(cart.Items, item)
	}
	cart.UpdatedAt = time.Now()
	data, err := json.Marshal(cart)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, guestKey(sessionID), data, guestTTL).Err()
}
