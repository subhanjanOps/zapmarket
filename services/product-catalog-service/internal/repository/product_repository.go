package repository

import (
	"database/sql"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db}
}

// func (pr *ProductRepository) CreateProduct(ctx context.Context, product *domain.Product) error {}

// func (pr *ProductRepository) GetProductByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
// }

// func (pr *ProductRepository) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
// }

// func (pr *ProductRepository) GetProductList(ctx context.Context, filters map[string]string) ([]*domain.Product, error) {
// }

// func (pr *ProductRepository) UpdateProduct(ctx context.Context, product *domain.Category) error {
// }

// func (pr *ProductRepository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
// }
