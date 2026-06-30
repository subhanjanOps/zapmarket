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
	// UpdatedAt in the WHERE clause provides optimistic concurrency control:
	// if another request updated the row between our read and this write,
	// updated_at will no longer match and rows affected will be 0 → 409.
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
			AND updated_at = $8
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
		product.UpdatedAt,
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
		return pkgerrors.NewConflict("PRODUCT_CONFLICT", "product was modified by another request; please retry")
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

// IncrSalesRank decrements the sales_rank column (lower = more popular) by 1.
// Initialises to 0 first if NULL, then decrements. Uses a large negative
// number as a sentinel: each confirmed order pushes the product higher.
func (pr *ProductRepository) IncrSalesRank(ctx context.Context, productID uuid.UUID) error {
	_, err := pr.db.ExecContext(ctx, `
		UPDATE products
		SET sales_rank = COALESCE(sales_rank, 0) - 1,
		    updated_at  = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, productID)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to increment sales rank", err)
	}
	return nil
}

// GetTrending returns up to limit products ordered by sales_rank ascending
// (more sales = lower/more-negative rank number).
func (pr *ProductRepository) UpsertCooccurrence(ctx context.Context, productA, productB uuid.UUID) error {
	// Always store pair in canonical order (smaller UUID first) to avoid duplicates.
	a, b := productA, productB
	if a.String() > b.String() {
		a, b = b, a
	}
	_, err := pr.db.ExecContext(ctx, `
		INSERT INTO product_cooccurrences (product_a, product_b, count, updated_at)
		VALUES ($1, $2, 1, NOW())
		ON CONFLICT (product_a, product_b) DO UPDATE
		  SET count = product_cooccurrences.count + 1, updated_at = NOW()
	`, a, b)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to upsert cooccurrence", err)
	}
	return nil
}

func (pr *ProductRepository) GetRecommendations(ctx context.Context, productID uuid.UUID, limit int) ([]*domain.Product, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	rows, err := pr.db.QueryContext(ctx, `
		SELECT p.id, p.name, p.description, p.seller_id, p.category_id, p.status, p.created_at, p.updated_at
		FROM product_cooccurrences c
		JOIN products p ON (
			CASE WHEN c.product_a = $1 THEN c.product_b ELSE c.product_a END = p.id
		)
		WHERE (c.product_a = $1 OR c.product_b = $1)
		  AND p.deleted_at IS NULL AND p.status = 'ACTIVE'
		ORDER BY c.count DESC
		LIMIT $2
	`, productID, limit)
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to get recommendations", err)
	}
	defer rows.Close()
	var out []*domain.Product
	for rows.Next() {
		p := &domain.Product{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.SellerID, &p.CategoryID, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to scan recommendation", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (pr *ProductRepository) GetForYou(ctx context.Context, categoryIDs []uuid.UUID, limit int) ([]*domain.Product, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if len(categoryIDs) == 0 {
		return pr.GetTrending(ctx, limit)
	}
	// Convert categoryIDs to a pq array.
	ids := make([]string, len(categoryIDs))
	for i, id := range categoryIDs {
		ids[i] = id.String()
	}
	// Build a parameterized ANY query.
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+1)
	args[0] = limit
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}
	query := fmt.Sprintf(`
		SELECT id, name, description, seller_id, category_id, status, created_at, updated_at
		FROM products
		WHERE category_id = ANY(ARRAY[%s]::uuid[])
		  AND deleted_at IS NULL AND status = 'ACTIVE'
		ORDER BY sales_rank ASC NULLS LAST, created_at DESC
		LIMIT $1
	`, strings.Join(placeholders, ","))
	rows, err := pr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to get for-you products", err)
	}
	defer rows.Close()
	var out []*domain.Product
	for rows.Next() {
		p := &domain.Product{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.SellerID, &p.CategoryID, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to scan for-you product", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (pr *ProductRepository) GetTrending(ctx context.Context, limit int) ([]*domain.Product, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := pr.db.QueryContext(ctx, `
		SELECT id, name, description, seller_id, category_id, status, created_at, updated_at
		FROM products
		WHERE sales_rank IS NOT NULL AND deleted_at IS NULL
		ORDER BY sales_rank ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to get trending products", err)
	}
	defer rows.Close()

	var out []*domain.Product
	for rows.Next() {
		p := &domain.Product{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.SellerID, &p.CategoryID, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to scan trending product", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
