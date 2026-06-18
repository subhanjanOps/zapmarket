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
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23505":
				return pkgerrors.NewConflict("PRODUCT_ALREADY_EXISTS", fmt.Sprintf("product with slug '%s' already exists", product.Slug))
			case "23503":
				return pkgerrors.NewValidation("INVALID_DATA", "category does not exist")
			case "23514":
				return pkgerrors.NewValidation("INVALID_DATA", "status must be one of DRAFT, ACTIVE, INACTIVE, ARCHIVED")
			}
		}
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to create product", err)
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
			return nil, pkgerrors.NewNotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return nil, pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to fetch product", err)
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
			return nil, pkgerrors.NewNotFound("PRODUCT_NOT_FOUND", "product not found")
		}
		return nil, pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to fetch product", err)
	}

	return product, nil
}

// productListWhere builds the shared WHERE clause + args for both the COUNT
// and SELECT queries in GetProductList, so the two queries can never drift
// apart (a common source of pagination bugs: filtering one way for the page
// and another for the total).
func productListWhere(filters *domain.ProductFilters) (string, []interface{}) {
	where := "WHERE deleted_at IS NULL"
	args := make([]interface{}, 0)
	argPos := 1

	if filters == nil {
		return where, args
	}

	if filters.CategoryID != nil {
		where += fmt.Sprintf(" AND category_id = $%d", argPos)
		args = append(args, *filters.CategoryID)
		argPos++
	}

	if filters.SellerID != nil {
		where += fmt.Sprintf(" AND seller_id = $%d", argPos)
		args = append(args, *filters.SellerID)
		argPos++
	}

	if filters.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filters.Status)
		argPos++
	}

	if filters.Slug != "" {
		where += fmt.Sprintf(" AND slug = $%d", argPos)
		args = append(args, filters.Slug)
		argPos++
	}

	if filters.Search != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argPos, argPos)
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	return where, args
}

func (pr *ProductRepository) GetProductList(ctx context.Context, filters *domain.ProductFilters) ([]*domain.Product, int64, error) {
	where, args := productListWhere(filters)

	var total int64
	countQuery := "SELECT COUNT(*) FROM products " + where
	if err := pr.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to count products", err)
	}

	sortBy := "created_at"
	sortOrder := "DESC"
	if filters != nil {
		switch filters.SortBy {
		case "name", "created_at", "updated_at":
			sortBy = filters.SortBy
		}
		if strings.EqualFold(filters.SortOrder, "ASC") {
			sortOrder = "ASC"
		}
	}

	query := fmt.Sprintf(
		"SELECT id, category_id, seller_id, name, slug, description, attributes, status, created_at, updated_at, deleted_at FROM products %s ORDER BY %s %s",
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

	rows, err := pr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to fetch products", err)
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
			return nil, 0, pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to scan product", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to iterate products", err)
	}

	return products, total, nil
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
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23505":
				return pkgerrors.NewConflict("PRODUCT_ALREADY_EXISTS", fmt.Sprintf("product with slug '%s' already exists", product.Slug))
			case "23503":
				return pkgerrors.NewValidation("INVALID_DATA", "category does not exist")
			}
		}
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to update product", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to get affected rows", err)
	}

	if rowsAffected == 0 {
		return pkgerrors.NewNotFound("PRODUCT_NOT_FOUND", "product not found")
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

	result, err := pr.db.ExecContext(ctx, query, id)
	if err != nil {
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to delete product", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to get affected rows", err)
	}

	if rowsAffected == 0 {
		return pkgerrors.NewNotFound("PRODUCT_NOT_FOUND", "product not found")
	}

	return nil
}
