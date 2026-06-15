package errors

import (
	"fmt"

	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
)

func CategoryNotFound(id string) error {
	return pkgerrors.NewNotFound("CATEGORY_NOT_FOUND", fmt.Sprintf("category %s not found", id))
}

func CategoryAlreadyExists(slug string) error {
	return pkgerrors.NewConflict("CATEGORY_ALREADY_EXISTS", fmt.Sprintf("category with slug %s already exists", slug))
}

func ProductNotFound(id string) error {
	return pkgerrors.NewNotFound("PRODUCT_NOT_FOUND", fmt.Sprintf("product %s not found", id))
}

func SKUNotFound(id string) error {
	return pkgerrors.NewNotFound("SKU_NOT_FOUND", fmt.Sprintf("sku %s not found", id))
}

func ImageNotFound(id string) error {
	return pkgerrors.NewNotFound("IMAGE_NOT_FOUND", fmt.Sprintf("image %s not found", id))
}
