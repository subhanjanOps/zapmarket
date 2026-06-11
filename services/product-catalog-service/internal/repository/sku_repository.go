package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	appErr "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/errors"
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
				return appErr.ConflictError(
					"sku code already exists",
					err,
				)
			}
		}

		return appErr.InternalError(
			"failed to create sku",
			err,
		)
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

	err := sr.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
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
			return nil, appErr.NotFoundError(
				"sku not found",
				err,
			)
		}

		return nil, appErr.InternalError(
			"failed to get sku",
			err,
		)
	}

	return sku, nil
}

// func (sr *SkuRepository) GetSkuBySlug(ctx context.Context, slug string) (*domain.SKU, error) {
// }

func (sr *SkuRepository) GetSkuList(
	ctx context.Context,
	filters *domain.SKUFilters,
) ([]*domain.SKU, error) {

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
		WHERE deleted_at IS NULL
	`

	args := make([]interface{}, 0)
	argPos := 1

	if filters != nil {
		if filters.ProductID != nil {
			query += fmt.Sprintf(
				" AND product_id = $%d",
				argPos,
			)

			args = append(args, *filters.ProductID)
			argPos++
		}

		if filters.SKUCode != nil {
			query += fmt.Sprintf(
				" AND sku_code ILIKE $%d",
				argPos,
			)

			args = append(
				args,
				"%"+*filters.SKUCode+"%",
			)

			argPos++
		}

		if filters.IsActive != nil {
			query += fmt.Sprintf(
				" AND is_active = $%d",
				argPos,
			)

			args = append(args, *filters.IsActive)
			argPos++
		}
	}

	sortBy := "created_at"
	sortOrder := "DESC"

	if filters != nil {
		switch filters.SortBy {
		case "created_at",
			"updated_at",
			"price_amount",
			"sku_code":
			sortBy = filters.SortBy
		}

		switch strings.ToUpper(filters.SortOrder) {
		case "ASC":
			sortOrder = "ASC"
		case "DESC":
			sortOrder = "DESC"
		}
	}

	query += fmt.Sprintf(
		" ORDER BY %s %s",
		sortBy,
		sortOrder,
	)

	limit := 50
	offset := 0

	if filters != nil {
		if filters.Limit > 0 {
			limit = filters.Limit
		}

		if filters.Offset >= 0 {
			offset = filters.Offset
		}
	}

	query += fmt.Sprintf(
		" LIMIT $%d OFFSET $%d",
		argPos,
		argPos+1,
	)

	args = append(args, limit, offset)

	rows, err := sr.db.QueryContext(
		ctx,
		query,
		args...,
	)

	if err != nil {
		return nil, appErr.InternalError(
			"failed to get sku list",
			err,
		)
	}

	defer rows.Close()

	var skus []*domain.SKU

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
			return nil, appErr.InternalError(
				"failed to scan sku",
				err,
			)
		}

		skus = append(skus, sku)
	}

	if err := rows.Err(); err != nil {
		return nil, appErr.InternalError(
			"failed while iterating skus",
			err,
		)
	}

	return skus, nil
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
				return appErr.ConflictError(
					"sku code already exists",
					err,
				)
			}
		}

		return appErr.InternalError(
			"failed to update sku",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return appErr.InternalError(
			"failed to get affected rows",
			err,
		)
	}

	if rowsAffected == 0 {
		return appErr.NotFoundError(
			"sku not found",
			err,
		)
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

	result, err := sr.db.ExecContext(
		ctx,
		query,
		id,
	)

	if err != nil {
		return appErr.InternalError(
			"failed to delete sku",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return appErr.InternalError(
			"failed to get affected rows",
			err,
		)
	}

	if rowsAffected == 0 {
		return appErr.NotFoundError(
			"sku not found",
			err,
		)
	}

	return nil
}
