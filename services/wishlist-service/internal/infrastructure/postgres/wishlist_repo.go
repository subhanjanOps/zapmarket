package postgres

import (
	"context"
	"database/sql"

	"github.com/zapmarket/zapmarket/services/wishlist-service/internal/application/usecases"
)

type WishlistRepo struct{ db *sql.DB }

func NewWishlistRepo(db *sql.DB) *WishlistRepo { return &WishlistRepo{db: db} }

func (r *WishlistRepo) CountByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM wishlist_items WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

func (r *WishlistRepo) Add(ctx context.Context, userID, productID, skuID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO wishlist_items (user_id, product_id, sku_id)
		VALUES ($1, $2, NULLIF($3, ''))
		ON CONFLICT (user_id, product_id) DO NOTHING`, userID, productID, skuID)
	return err
}

func (r *WishlistRepo) List(ctx context.Context, userID string) ([]usecases.WishlistItemDTO, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT product_id, COALESCE(sku_id::text, '') FROM wishlist_items
		WHERE user_id = $1 ORDER BY added_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []usecases.WishlistItemDTO{}
	for rows.Next() {
		var item usecases.WishlistItemDTO
		if err := rows.Scan(&item.ProductID, &item.SKUID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *WishlistRepo) Remove(ctx context.Context, userID, productID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM wishlist_items WHERE user_id = $1 AND product_id = $2`, userID, productID)
	return err
}

func (r *WishlistRepo) Clear(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM wishlist_items WHERE user_id = $1`, userID)
	return err
}
