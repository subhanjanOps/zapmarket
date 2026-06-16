package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
)

type SkuRepository struct {
	db *sql.DB
}

func NewSkuRepository(db *sql.DB) *SkuRepository {
	return &SkuRepository{db}
}

func (sr *SkuRepository) CreateSku(
	ctx context.Context,
	sku *domain.SKU,
) error {
	query := `
		INSERT INTO skus (
			id,
			product_id,
			sku_code,
			variant_attrs,
			price_amount,
			compare_price,
			currency,
			weight_grams,
			is_active,
			created_at,
			updated_at
		)
		VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW()
		)
	`

	_, err := sr.db.ExecContext(
		ctx,
		query,
		sku.ID,
		sku.ProductID,
		sku.SKUCode,
		sku.VariantAttrs,
		sku.PriceAmount,
		sku.ComparePrice,
		sku.Currency,
		sku.WeightGrams,
		sku.IsActive,
	)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23505":
				return pkgerrors.NewConflict("SKU_ALREADY_EXISTS", "sku code already exists")
			}
		}
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to create sku", err)
	}

	return nil
}

func (sr *SkuRepository) GetSkuByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.SKU, error) {
	query := `
		SELECT
			id,
			product_id,
			sku_code,
			variant_attrs,
			price_amount,
			compare_price,
			currency,
			weight_grams,
			is_active,
			created_at,
			updated_at,
			deleted_at
		FROM skus
		WHERE id = $1
		AND deleted_at IS NULL
	`

	sku := &domain.SKU{}

	err := sr.db.QueryRowContext(ctx, query, id).Scan(
		&sku.ID,
		&sku.ProductID,
		&sku.SKUCode,
		&sku.VariantAttrs,
		&sku.PriceAmount,
		&sku.ComparePrice,
		&sku.Currency,
		&sku.WeightGrams,
		&sku.IsActive,
		&sku.CreatedAt,
		&sku.UpdatedAt,
		&sku.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgerrors.NewNotFound("SKU_NOT_FOUND", "sku not found")
		}
		return nil, pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to get sku", err)
	}

	return sku, nil
}

// skuListWhere builds the shared WHERE clause + args for both the COUNT and
// SELECT queries in GetSkuList, so the two queries can never drift apart (a
// common source of pagination bugs: filtering one way for the page and
// another for the total).
func skuListWhere(filters *domain.SKUFilters) (string, []interface{}) {
	where := "WHERE deleted_at IS NULL"
	args := make([]interface{}, 0)
	argPos := 1

	if filters == nil {
		return where, args
	}

	if filters.ProductID != nil {
		where += fmt.Sprintf(" AND product_id = $%d", argPos)
		args = append(args, *filters.ProductID)
		argPos++
	}

	if filters.SKUCode != nil {
		where += fmt.Sprintf(" AND sku_code ILIKE $%d", argPos)
		args = append(args, "%"+*filters.SKUCode+"%")
		argPos++
	}

	if filters.IsActive != nil {
		where += fmt.Sprintf(" AND is_active = $%d", argPos)
		args = append(args, *filters.IsActive)
		argPos++
	}

	return where, args
}

func (sr *SkuRepository) GetSkuList(
	ctx context.Context,
	filters *domain.SKUFilters,
) ([]*domain.SKU, int64, error) {
	where, args := skuListWhere(filters)

	var total int64
	countQuery := "SELECT COUNT(*) FROM skus " + where
	if err := sr.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to count skus", err)
	}

	sortBy := "created_at"
	sortOrder := "DESC"
	if filters != nil {
		switch filters.SortBy {
		case "created_at", "updated_at", "price_amount", "sku_code":
			sortBy = filters.SortBy
		}
		if strings.EqualFold(filters.SortOrder, "ASC") {
			sortOrder = "ASC"
		}
	}

	query := fmt.Sprintf(
		"SELECT id, product_id, sku_code, variant_attrs, price_amount, compare_price, currency, weight_grams, is_active, created_at, updated_at, deleted_at FROM skus %s ORDER BY %s %s",
		where, sortBy, sortOrder,
	)

	argPos := len(args) + 1
	limit := domain.DefaultPageSize
	if filters != nil && filters.Limit > 0 {
		limit = filters.Limit
	}
	offset := 0
	if filters != nil && filters.Offset > 0 {
		offset = filters.Offset
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := sr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to get sku list", err)
	}
	defer rows.Close()

	skus := make([]*domain.SKU, 0)

	for rows.Next() {
		sku := &domain.SKU{}

		err := rows.Scan(
			&sku.ID,
			&sku.ProductID,
			&sku.SKUCode,
			&sku.VariantAttrs,
			&sku.PriceAmount,
			&sku.ComparePrice,
			&sku.Currency,
			&sku.WeightGrams,
			&sku.IsActive,
			&sku.CreatedAt,
			&sku.UpdatedAt,
			&sku.DeletedAt,
		)

		if err != nil {
			return nil, 0, pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to scan sku", err)
		}

		skus = append(skus, sku)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed while iterating skus", err)
	}

	return skus, total, nil
}

func (sr *SkuRepository) UpdateSku(
	ctx context.Context,
	sku *domain.SKU,
) error {
	query := `
		UPDATE skus
		SET
			product_id = $1,
			sku_code = $2,
			variant_attrs = $3,
			price_amount = $4,
			compare_price = $5,
			currency = $6,
			weight_grams = $7,
			is_active = $8,
			updated_at = NOW()
		WHERE id = $9
		AND deleted_at IS NULL
	`

	result, err := sr.db.ExecContext(
		ctx,
		query,
		sku.ProductID,
		sku.SKUCode,
		sku.VariantAttrs,
		sku.PriceAmount,
		sku.ComparePrice,
		sku.Currency,
		sku.WeightGrams,
		sku.IsActive,
		sku.ID,
	)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23505":
				return pkgerrors.NewConflict("SKU_ALREADY_EXISTS", "sku code already exists")
			}
		}
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to update sku", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to get affected rows", err)
	}

	if rowsAffected == 0 {
		return pkgerrors.NewNotFound("SKU_NOT_FOUND", "sku not found")
	}

	return nil
}

func (sr *SkuRepository) DeleteSku(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		UPDATE skus
		SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
		AND deleted_at IS NULL
	`

	result, err := sr.db.ExecContext(ctx, query, id)
	if err != nil {
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to delete sku", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to get affected rows", err)
	}

	if rowsAffected == 0 {
		return pkgerrors.NewNotFound("SKU_NOT_FOUND", "sku not found")
	}

	return nil
}
