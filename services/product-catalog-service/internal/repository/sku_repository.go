package repository

import (
	"database/sql"
)

type SkuRepository struct {
	db *sql.DB
}

func NewSkuRepository(db *sql.DB) *SkuRepository {
	return &SkuRepository{db}
}

// func (sr *SkuRepository) CreateSku(ctx context.Context, sku *domain.SKU) error {
// }

// func (sr *SkuRepository) GetSkuByID(ctx context.Context, id uuid.UUID) (*domain.SKU, error) {
// }

// func (sr *SkuRepository) GetSkuBySlug(ctx context.Context, slug string) (*domain.SKU, error) {
// }

// func (sr *SkuRepository) GetSkuList(ctx context.Context, filters map[string]string) ([]*domain.SKU, error) {
// }

// func (sr *SkuRepository) UpdateSku(ctx context.Context, product *domain.SKU) error {
// }

// func (sr *SkuRepository) DeleteSku(ctx context.Context, id uuid.UUID) error {
// }
