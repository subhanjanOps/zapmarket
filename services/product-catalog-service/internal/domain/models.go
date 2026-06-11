package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID       uuid.UUID  `json:"id"`
	Name     string     `json:"name"`
	Slug     string     `json:"slug"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type CategoryFilters struct {
	ParentID *uuid.UUID
}

type Product struct {
	ID          uuid.UUID       `json:"id"`
	CategoryID  uuid.UUID       `json:"category_id"`
	SellerID    uuid.UUID       `json:"seller_id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description *string         `json:"description,omitempty"`
	Attributes  json.RawMessage `json:"attributes,omitempty"`
	Status      ProductStatus   `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty"`
}

type ProductFilters struct {
	CategoryID *uuid.UUID
	SellerID   *uuid.UUID
	Status     string
	Slug       string
	Search     string

	Limit  int
	Offset int

	SortBy    string
	SortOrder string
}

type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
	ProductStatusArchived ProductStatus = "archived"
)

type SKU struct {
	ID           uuid.UUID       `json:"id"`
	ProductID    uuid.UUID       `json:"product_id"`
	SKUCode      string          `json:"sku_code"`
	VariantAttrs json.RawMessage `json:"variant_attributes,omitempty"`
	PriceAmount  int64           `json:"price_amount"`
	ComparePrice *int64          `json:"compare_price,omitempty"`
	Currency     string          `json:"currency"`
	WeightGrams  *int32          `json:"weight_grams,omitempty"`
	IsActive     bool            `json:"is_active"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    *time.Time      `json:"deleted_at,omitempty"`
}

type SKUFilters struct {
	ProductID *uuid.UUID
	SKUCode   *string
	IsActive  *bool
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}
type ProductImage struct {
	ID        uuid.UUID  `json:"id"`
	ProductID uuid.UUID  `json:"product_id"`
	SKUId     *uuid.UUID `json:"sku_id,omitempty"`
	URL       string     `json:"url"`
	Position  int        `json:"position"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
