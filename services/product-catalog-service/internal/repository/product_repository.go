package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	appErr "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/errors"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db}
}

func (pr *ProductRepository) CreateProduct(
	ctx context.Context,
	product *domain.Product,
) error {
	query := `
		INSERT INTO products (
			id,
			category_id,
			seller_id,
			name,
			slug,
			description,
			attributes,
			status,
			created_at,
			updated_at
		)
		VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW()
		);
	`

	_, err := pr.db.ExecContext(
		ctx,
		query,
		product.ID,
		product.CategoryID,
		product.SellerID,
		product.Name,
		product.Slug,
		product.Description,
		product.Attributes,
		product.Status,
	)

	if err != nil {
		return appErr.InternalError(
			"failed to create product",
			err,
		)
	}

	return nil
}

func (pr *ProductRepository) GetProductByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Product, error) {
	query := `
		SELECT
			id,
			category_id,
			seller_id,
			name,
			slug,
			description,
			attributes,
			status,
			created_at,
			updated_at,
			deleted_at
		FROM products
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	product := &domain.Product{}

	err := pr.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.CategoryID,
		&product.SellerID,
		&product.Name,
		&product.Slug,
		&product.Description,
		&product.Attributes,
		&product.Status,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErr.NotFoundError(
				"product not found",
				err,
			)
		}

		return nil, appErr.InternalError(
			"failed to fetch product",
			err,
		)
	}

	return product, nil
}

func (pr *ProductRepository) GetProductBySlug(
	ctx context.Context,
	slug string,
) (*domain.Product, error) {
	query := `
		SELECT
			id,
			category_id,
			seller_id,
			name,
			slug,
			description,
			attributes,
			status,
			created_at,
			updated_at,
			deleted_at
		FROM products
		WHERE slug = $1
			AND deleted_at IS NULL;
	`

	product := &domain.Product{}

	err := pr.db.QueryRowContext(ctx, query, slug).Scan(
		&product.ID,
		&product.CategoryID,
		&product.SellerID,
		&product.Name,
		&product.Slug,
		&product.Description,
		&product.Attributes,
		&product.Status,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErr.NotFoundError(
				"product not found",
				err,
			)
		}

		return nil, appErr.InternalError(
			"failed to fetch product",
			err,
		)
	}

	return product, nil
}

func (pr *ProductRepository) GetProductList(ctx context.Context, filters *domain.ProductFilters) ([]*domain.Product, error) {
	query := `
		SELECT id, category_id, seller_id, name, slug, description, attributes, status, created_at, updated_at, deleted_at
		FROM products
		WHERE deleted_at IS NULL
	`

	args := make([]interface{}, 0)
	argPos := 1

	if filters != nil {
		if filters.CategoryID != nil {
			query += fmt.Sprintf(" AND category_id = $%d", argPos)
			args = append(args, *filters.CategoryID)
			argPos++
		}

		if filters.SellerID != nil {
			query += fmt.Sprintf(" AND seller_id = $%d", argPos)
			args = append(args, *filters.SellerID)
			argPos++
		}

		if filters.Status != "" {
			query += fmt.Sprintf(" AND status = $%d", argPos)
			args = append(args, filters.Status)
			argPos++
		}

		if filters.Slug != "" {
			query += fmt.Sprintf(" AND slug = $%d", argPos)
			args = append(args, filters.Slug)
			argPos++
		}

		if filters.Search != "" {
			query += fmt.Sprintf(`
				AND (
					name ILIKE $%d
					OR description ILIKE $%d
				)
			`, argPos, argPos)

			args = append(args, "%"+filters.Search+"%")
			argPos++
		}
	}

	sortBy := "created_at"
	sortOrder := "DESC"

	if filters != nil {
		switch filters.SortBy {
		case "name", "created_at", "updated_at":
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

	if filters != nil && filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filters.Limit)
		argPos++
	}

	if filters != nil && filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filters.Offset)
		argPos++
	}

	rows, err := pr.db.QueryContext(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, appErr.InternalError(
			"failed to fetch products",
			err,
		)
	}
	defer rows.Close()

	products := make([]*domain.Product, 0)

	for rows.Next() {
		product := &domain.Product{}

		err := rows.Scan(
			&product.ID,
			&product.CategoryID,
			&product.SellerID,
			&product.Name,
			&product.Slug,
			&product.Description,
			&product.Attributes,
			&product.Status,
			&product.CreatedAt,
			&product.UpdatedAt,
			&product.DeletedAt,
		)
		if err != nil {
			return nil, appErr.InternalError(
				"failed to scan product",
				err,
			)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, appErr.InternalError(
			"failed to iterate products",
			err,
		)
	}

	return products, nil
}

func (pr *ProductRepository) UpdateProduct(ctx context.Context, product *domain.Product) error {
	query := `
		UPDATE products
		SET
			category_id = $1,
			name = $2,
			slug = $3,
			description = $4,
			attributes = $5,
			status = $6,
			updated_at = NOW()
		WHERE id = $7
			AND deleted_at IS NULL;
	`

	result, err := pr.db.ExecContext(
		ctx,
		query,
		product.CategoryID,
		product.Name,
		product.Slug,
		product.Description,
		product.Attributes,
		product.Status,
		product.ID,
	)

	if err != nil {
		return appErr.InternalError(
			"failed to update product",
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
			"product not found",
			nil,
		)
	}

	return nil
}
func (pr *ProductRepository) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE products
		SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	result, err := pr.db.ExecContext(
		ctx,
		query,
		id,
	)

	if err != nil {
		return appErr.InternalError(
			"failed to delete product",
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
			"product not found",
			nil,
		)
	}

	return nil
}
