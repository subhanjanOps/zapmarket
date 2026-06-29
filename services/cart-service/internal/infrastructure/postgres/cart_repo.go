package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/cart-service/internal/domain"
)

type CartRepo struct{ db *sql.DB }

func NewCartRepo(db *sql.DB) *CartRepo { return &CartRepo{db: db} }

func (r *CartRepo) GetCart(ctx context.Context, userID string) (*domain.Cart, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT sku_id, product_id, product_name, variant_attrs, quantity,
		       price_at_add, currency, COALESCE(image_url,''), added_at
		FROM cart_items WHERE user_id = $1 ORDER BY added_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cart := &domain.Cart{UserID: userID, UpdatedAt: time.Now()}
	for rows.Next() {
		var item domain.CartItem
		var attrsJSON []byte
		if err := rows.Scan(
			&item.SKUID, &item.ProductID, &item.ProductName, &attrsJSON,
			&item.Quantity, &item.PriceAtAdd, &item.Currency, &item.ImageURL, &item.AddedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(attrsJSON, &item.VariantAttrs)
		cart.Items = append(cart.Items, item)
	}
	return cart, rows.Err()
}

func (r *CartRepo) UpsertItem(ctx context.Context, userID string, item domain.CartItem) error {
	attrsJSON, _ := json.Marshal(item.VariantAttrs)
	uid, _ := uuid.Parse(userID)
	skuID, _ := uuid.Parse(item.SKUID)
	productID, _ := uuid.Parse(item.ProductID)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cart_items (user_id, sku_id, product_id, product_name, variant_attrs, quantity, price_at_add, currency, image_url, added_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())
		ON CONFLICT (user_id, sku_id) DO UPDATE
		SET quantity=$6, price_at_add=$7, updated_at=NOW()`,
		uid, skuID, productID, item.ProductName, attrsJSON,
		item.Quantity, item.PriceAtAdd, item.Currency, item.ImageURL)
	return err
}

func (r *CartRepo) RemoveItem(ctx context.Context, userID, skuID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cart_items WHERE user_id=$1 AND sku_id=$2`, userID, skuID)
	return err
}

func (r *CartRepo) ClearCart(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cart_items WHERE user_id=$1`, userID)
	return err
}
