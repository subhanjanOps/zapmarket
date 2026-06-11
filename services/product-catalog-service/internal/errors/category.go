package errors

import "fmt"

func CategoryNotFound(id string) error {
	return &AppError{
		Type:    NotFound,
		Code:    "CATEGORY_NOT_FOUND",
		Message: fmt.Sprintf("category %s not found", id),
	}
}

func CategoryAlreadyExists(slug string) error {
	return &AppError{
		Type:    Conflict,
		Code:    "CATEGORY_ALREADY_EXISTS",
		Message: fmt.Sprintf("category with slug %s already exists", slug),
	}
}

func ProductNotFound(id string) error {
	return &AppError{
		Type:    NotFound,
		Code:    "PRODUCT_NOT_FOUND",
		Message: fmt.Sprintf("product %s not found", id),
	}
}

func SKUNotFound(id string) error {
	return &AppError{
		Type:    NotFound,
		Code:    "SKU_NOT_FOUND",
		Message: fmt.Sprintf("sku %s not found", id),
	}
}
