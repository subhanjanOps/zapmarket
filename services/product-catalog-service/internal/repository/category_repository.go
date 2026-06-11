package repository

import (
	"database/sql"
)

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db}
}

// func (cr *CategoryRepository) CreateCategory(ctx context.Context, category *domain.Category) error {
// }

// func (cr *CategoryRepository) GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
// }

// func (cr *CategoryRepository) GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error) {
// }

// func (cr *CategoryRepository) GetCategoryList(ctx context.Context, filters map[string]string) ([]*domain.Category, error) {
// }

// func (cr *CategoryRepository) UpdateCategory(ctx context.Context, product *domain.Category) error {
// }

// func (cr *CategoryRepository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
// }
