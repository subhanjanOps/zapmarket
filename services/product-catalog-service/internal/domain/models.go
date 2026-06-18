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

// BulkCategoryInput is the service-layer input for one row of a bulk import.
// ParentName is resolved server-side to a ParentID before insertion.
type BulkCategoryInput struct {
	Name       string
	Slug       string
	ParentID   *uuid.UUID
	ParentName *string
}

// DefaultPageSize and MaxPageSize bound every list endpoint's limit/offset
// query params so a client can't force an unbounded table scan.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type CategoryFilters struct {
	ParentID *uuid.UUID
	RootOnly bool
	Search   string

	Limit  int
	Offset int

	SortBy    string
	SortOrder string
}

type Product struct {
	ID          uuid.UUID       `json:"id"`
	CategoryID  uuid.UUID       `json:"category_id"`
	SellerID    uuid.UUID       `json:"seller_id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description *string         `json:"description,omitempty"`
	Attributes  json.RawMessage `json:"attributes,omitempty" swaggertype:"object"`
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
	ProductStatusDraft    ProductStatus = "DRAFT"
	ProductStatusActive   ProductStatus = "ACTIVE"
	ProductStatusInactive ProductStatus = "INACTIVE"
	ProductStatusArchived ProductStatus = "ARCHIVED"
)

type SKU struct {
	ID           uuid.UUID       `json:"id"`
	ProductID    uuid.UUID       `json:"product_id"`
	SKUCode      string          `json:"sku_code"`
	VariantAttrs json.RawMessage `json:"variant_attributes,omitempty" swaggertype:"object"`
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
	ID uuid.UUID `json:"id"`
	// ObjectKey is the MinIO/S3 object key this image was uploaded to (empty
	// for any legacy rows that only ever had an externally-supplied URL).
	// It's the source of truth for storage operations (deletion); URL is a
	// derived, display-only value built from it.
	ObjectKey string     `json:"-"`
	ProductID uuid.UUID  `json:"product_id"`
	SKUId     *uuid.UUID `json:"sku_id,omitempty"`
	URL       string     `json:"url"`
	Position  int        `json:"position"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
