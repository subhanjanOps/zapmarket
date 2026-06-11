package repository

import (
	"database/sql"
)

type ProductImageRepository struct {
	db *sql.DB
}

func NewProductImageRepository(db *sql.DB) *ProductImageRepository {
	return &ProductImageRepository{db}
}

// func (pir *ProductImageRepository) createProductImage(ctx context.Context, prdImage domain.ProductImage) error {
// }

// func (pir *ProductImageRepository) GetImageByProductID(ctx context.Context, productID uuid.UUID) ([]*domain.ProductImage, error) {
// }
