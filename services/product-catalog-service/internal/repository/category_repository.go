package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	appErr "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/errors"
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

			// unique_violation
			case "23505":
				return appErr.ConflictError(
					fmt.Sprintf(
						"category with slug '%s' already exists",
						category.Slug,
					),
					err,
				)
			}
		}

		return appErr.DatabaseError(
			"failed to create category",
			err,
		)
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
			return nil, appErr.DatabaseError(
				"category not found",
				err,
			)
		}

		return nil, appErr.DatabaseError(
			"failed to get category",
			err,
		)
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
			return nil, appErr.DatabaseError(
				"category not found",
				err,
			)
		}
		return nil, appErr.DatabaseError(
			"failed to get category",
			err,
		)
	}

	return category, nil

}

func (cr *CategoryRepository) GetCategoryList(ctx context.Context, filters map[string]string, limit int, offset int) ([]*domain.Category, error) {
	query := `
		SELECT id, name, slug, parent_id, created_at, updated_at
		FROM categories
		WHERE deleted_at IS NULL AND parent_id IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2;
	`

	categories := make([]*domain.Category, 0)

	rows, err := cr.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, appErr.DatabaseError(
			"failed to fetch categories",
			err,
		)
	}
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
			return nil, appErr.DatabaseError(
				"failed to scan category",
				err,
			)
		}

		categories = append(categories, &category)
	}

	return categories, nil
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

			// unique_violation
			case "23505":
				return appErr.ConflictError(
					"category slug already exists",
					err,
				)

			// foreign_key_violation
			case "23503":
				return appErr.ValidationError(
					"parent category does not exist",
					err,
				)
			}
		}

		return appErr.DatabaseError(
			"failed to update category",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return appErr.DatabaseError(
			"failed to get affected rows",
			err,
		)
	}

	if rowsAffected == 0 {
		return appErr.NotFoundError(
			"category not found",
			nil,
		)
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
		return appErr.InternalError(
			"failed to delete category",
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
			"category not found",
			nil,
		)
	}

	return nil
}
