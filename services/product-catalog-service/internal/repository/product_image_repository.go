package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	appErr "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/errors"
)

type ProductImageRepository struct {
	db *sql.DB
}

func NewProductImageRepository(db *sql.DB) *ProductImageRepository {
	return &ProductImageRepository{db}
}

func (pir *ProductImageRepository) CreateProductImage(
	ctx context.Context,
	prdImage *domain.ProductImage,
) error {

	query := `
		INSERT INTO product_images (
			id,
			product_id,
			sku_id,
			url,
			position,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			COALESCE(
				(
					SELECT MAX(position) + 1
					FROM product_images
					WHERE
						product_id = $2
						AND deleted_at IS NULL
				),
				1
			),
			NOW(),
			NOW()
		)
	`

	_, err := pir.db.ExecContext(
		ctx,
		query,
		prdImage.ID,
		prdImage.ProductID,
		prdImage.SKUId,
		prdImage.URL,
	)

	if err != nil {
		return appErr.InternalError(
			"failed to create product image",
			err,
		)
	}

	return nil
}

func (pir *ProductImageRepository) GetImageByProductID(
	ctx context.Context,
	productID uuid.UUID,
) ([]*domain.ProductImage, error) {

	query := `
		SELECT
			id,
			product_id,
			sku_id,
			url,
			position,
			created_at,
			updated_at,
			deleted_at
		FROM product_images
		WHERE
			product_id = $1
			AND deleted_at IS NULL
		ORDER BY position ASC
	`

	rows, err := pir.db.QueryContext(
		ctx,
		query,
		productID,
	)

	if err != nil {
		return nil, appErr.InternalError(
			"failed to fetch product images",
			err,
		)
	}

	defer rows.Close()

	var images []*domain.ProductImage

	for rows.Next() {
		image := &domain.ProductImage{}

		err := rows.Scan(
			&image.ID,
			&image.ProductID,
			&image.SKUId,
			&image.URL,
			&image.Position,
			&image.CreatedAt,
			&image.UpdatedAt,
			&image.DeletedAt,
		)

		if err != nil {
			return nil, appErr.InternalError(
				"failed to scan product image",
				err,
			)
		}

		images = append(images, image)
	}

	if err := rows.Err(); err != nil {
		return nil, appErr.InternalError(
			"failed while iterating product images",
			err,
		)
	}

	return images, nil
}

func (pir *ProductImageRepository) GetImageBySKUID(
	ctx context.Context,
	skuID uuid.UUID,
) ([]*domain.ProductImage, error) {

	query := `
		SELECT
			id,
			product_id,
			sku_id,
			url,
			position,
			created_at,
			updated_at,
			deleted_at
		FROM product_images
		WHERE
			sku_id = $1
			AND deleted_at IS NULL
		ORDER BY position ASC
	`

	rows, err := pir.db.QueryContext(ctx, query, skuID)
	if err != nil {
		return nil, appErr.InternalError(
			"failed to fetch sku images",
			err,
		)
	}
	defer rows.Close()

	var images []*domain.ProductImage

	for rows.Next() {
		img := &domain.ProductImage{}

		err := rows.Scan(
			&img.ID,
			&img.ProductID,
			&img.SKUId,
			&img.URL,
			&img.Position,
			&img.CreatedAt,
			&img.UpdatedAt,
			&img.DeletedAt,
		)
		if err != nil {
			return nil, appErr.InternalError(
				"failed to scan product image",
				err,
			)
		}

		images = append(images, img)
	}

	if err := rows.Err(); err != nil {
		return nil, appErr.InternalError(
			"failed while iterating sku images",
			err,
		)
	}

	return images, nil
}

func (pir *ProductImageRepository) UpdateProductImagePosition(
	ctx context.Context,
	id uuid.UUID,
	position int,
) error {

	query := `
		UPDATE product_images
		SET
			position = $1,
			updated_at = NOW()
		WHERE
			id = $2
			AND deleted_at IS NULL
	`

	result, err := pir.db.ExecContext(
		ctx,
		query,
		position,
		id,
	)

	if err != nil {
		return appErr.InternalError(
			"failed to update image position",
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
			"product image not found",
			nil,
		)
	}

	return nil
}

func (pir *ProductImageRepository) DeleteProductImage(
	ctx context.Context,
	id uuid.UUID,
) error {

	query := `
		UPDATE product_images
		SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE
			id = $1
			AND deleted_at IS NULL
	`

	result, err := pir.db.ExecContext(
		ctx,
		query,
		id,
	)

	if err != nil {
		return appErr.InternalError(
			"failed to delete product image",
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
			"product image not found",
			nil,
		)
	}

	return nil
}
