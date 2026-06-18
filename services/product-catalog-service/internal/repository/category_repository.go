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

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db}
}

func (cr *CategoryRepository) CreateCategory(
	ctx context.Context,
	category *domain.Category,
) error {

	query := `
		INSERT INTO categories
			(id, name, slug, parent_id, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, NOW(), NOW())
	`

	_, err := cr.db.ExecContext(
		ctx,
		query,
		category.ID,
		category.Name,
		category.Slug,
		category.ParentID,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505":
				return pkgerrors.NewConflict("CATEGORY_ALREADY_EXISTS", fmt.Sprintf("category with slug '%s' already exists", category.Slug))
			}
		}
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to create category", err)
	}

	return nil
}

func (cr *CategoryRepository) BulkCreateCategories(ctx context.Context, categories []*domain.Category) error {
	if len(categories) == 0 {
		return nil
	}

	tx, err := cr.db.BeginTx(ctx, nil)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to begin transaction", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO categories (id, name, slug, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (slug) DO NOTHING
	`)
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to prepare statement", err)
	}
	defer stmt.Close()

	for _, cat := range categories {
		if _, err := stmt.ExecContext(ctx, cat.ID, cat.Name, cat.Slug, cat.ParentID); err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				return pkgerrors.NewConflict("CATEGORY_ALREADY_EXISTS", fmt.Sprintf("category with slug '%s' already exists", cat.Slug))
			}
			return pkgerrors.NewInternal("DATABASE_ERROR", "failed to insert category", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to commit transaction", err)
	}
	return nil
}

func (cr *CategoryRepository) GetCategoryByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Category, error) {

	query := `
		SELECT
			id,
			name,
			slug,
			parent_id,
			created_at,
			updated_at
		FROM categories
		WHERE id = $1
			AND deleted_at IS NULL
	`

	category := &domain.Category{}

	err := cr.db.QueryRowContext(ctx, query, id).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.ParentID,
		&category.CreatedAt,
		&category.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgerrors.NewNotFound("CATEGORY_NOT_FOUND", "category not found")
		}
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to get category", err)
	}

	return category, nil
}

func (cr *CategoryRepository) GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	query := `
	SELECT id, name, slug, parent_id, created_at, updated_at
	FROM categories
	WHERE slug = $1 AND deleted_at IS NULL
	`

	category := &domain.Category{}
	err := cr.db.QueryRowContext(ctx, query, slug).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.ParentID,
		&category.CreatedAt,
		&category.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgerrors.NewNotFound("CATEGORY_NOT_FOUND", "category not found")
		}
		return nil, pkgerrors.NewInternal("DATABASE_ERROR", "failed to get category", err)
	}

	return category, nil
}

// categoryListWhere builds the shared WHERE clause + args for both the
// COUNT and SELECT queries in GetCategoryList, so the two queries can never
// drift apart (a common source of pagination bugs: filtering one way for
// the page and another for the total).
func categoryListWhere(filters *domain.CategoryFilters) (string, []interface{}) {
	where := "WHERE deleted_at IS NULL"
	args := make([]interface{}, 0)
	argPos := 1

	if filters == nil {
		return where, args
	}

	if filters.ParentID != nil {
		where += fmt.Sprintf(" AND parent_id = $%d", argPos)
		args = append(args, *filters.ParentID)
		argPos++
	}

	if filters.Search != "" {
		where += fmt.Sprintf(" AND name ILIKE $%d", argPos)
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	return where, args
}

func (cr *CategoryRepository) GetCategoryList(ctx context.Context, filters *domain.CategoryFilters) ([]*domain.Category, int64, error) {
	where, args := categoryListWhere(filters)

	var total int64
	countQuery := "SELECT COUNT(*) FROM categories " + where
	if err := cr.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to count categories", err)
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
		"SELECT id, name, slug, parent_id, created_at, updated_at FROM categories %s ORDER BY %s %s",
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

	categories := make([]*domain.Category, 0)

	rows, err := cr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to fetch categories", err)
	}
	defer rows.Close()

	for rows.Next() {
		var category domain.Category

		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.ParentID,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to scan category", err)
		}

		categories = append(categories, &category)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, pkgerrors.NewInternal("DATABASE_ERROR", "failed to iterate categories", err)
	}

	return categories, total, nil
}

func (cr *CategoryRepository) UpdateCategory(
	ctx context.Context,
	category *domain.Category,
) error {

	query := `
		UPDATE categories
		SET
			name = $1,
			slug = $2,
			parent_id = $3,
			updated_at = NOW()
		WHERE id = $4
			AND deleted_at IS NULL
	`

	result, err := cr.db.ExecContext(
		ctx,
		query,
		category.Name,
		category.Slug,
		category.ParentID,
		category.ID,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23505":
				return pkgerrors.NewConflict("CATEGORY_ALREADY_EXISTS", "category slug already exists")
			case "23503":
				return pkgerrors.NewValidation("INVALID_DATA", "parent category does not exist")
			}
		}
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to update category", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.NewInternal("DATABASE_ERROR", "failed to get affected rows", err)
	}

	if rowsAffected == 0 {
		return pkgerrors.NewNotFound("CATEGORY_NOT_FOUND", "category not found")
	}

	return nil
}

func (cr *CategoryRepository) DeleteCategory(
	ctx context.Context,
	id uuid.UUID,
) error {
	query := `
		UPDATE categories
		SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL;
	`

	result, err := cr.db.ExecContext(ctx, query, id)
	if err != nil {
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to delete category", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.NewInternal("INTERNAL_SERVER_ERROR", "failed to get affected rows", err)
	}

	if rowsAffected == 0 {
		return pkgerrors.NewNotFound("CATEGORY_NOT_FOUND", "category not found")
	}

	return nil
}
