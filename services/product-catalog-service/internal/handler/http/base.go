package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/pkg/httpx"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/authctx"
	domainerrors "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/errors"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/service"
)

// Response documents the standard API envelope shape for swag/swagger
// generation (`swag init` needs a concrete, exported type to reference in
// `@Success`/`@Failure` annotations). It is not used at runtime — actual
// responses are written by httpx.JSON/Success/Error in pkg/httpx, whose
// envelope this type must stay in sync with.
type Response struct {
	Success bool        `json:"success"`
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON writes a JSON response. Thin alias over pkg/httpx so handlers in this
// package don't need to import httpx directly.
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	httpx.JSON(w, statusCode, data)
}

// SuccessResponse returns a successful response, delegating to pkg/httpx for
// the standard envelope shape shared across services.
func SuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	httpx.Success(w, statusCode, data)
}

// ErrorResponse returns an error response, delegating to pkg/httpx for the
// standard envelope shape shared across services.
func ErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	httpx.Error(w, statusCode, code, message)
}

// HandleError maps an AppError to the appropriate HTTP status and writes the response.
func HandleError(w http.ResponseWriter, err error) {
	pkgerrors.HandleHTTP(w, err)
}

// DecodeJSON decodes JSON request body
func DecodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// assertOwnership fetches the product by ID and verifies that the
// authenticated user either owns it (product.SellerID == user.Id) or is an
// admin. It returns a typed 403 Forbidden error on mismatch, propagating the
// underlying lookup error (e.g. 404) when the product cannot be fetched.
func assertOwnership(ctx context.Context, productSvc service.ProductService, productID uuid.UUID, user interface {
	GetId() string
	GetRole() string
}) error {
	if user == nil {
		return pkgerrors.NewUnauthorized("UNAUTHENTICATED", "authentication required")
	}

	product, err := productSvc.GetProductByID(ctx, productID)
	if err != nil {
		return err
	}

	if user.GetRole() == "admin" {
		return nil
	}

	if product.SellerID.String() != user.GetId() {
		return domainerrors.Forbidden()
	}

	return nil
}

// requireUser resolves the authenticated user from the request context,
// returning a typed 401 error when absent.
func requireUser(r *http.Request) (*authUser, error) {
	u := authctx.UserFromContext(r.Context())
	if u == nil {
		return nil, pkgerrors.NewUnauthorized("UNAUTHENTICATED", "authentication required")
	}
	return &authUser{u.GetId(), u.GetRole()}, nil
}

// authUser is a minimal value object satisfying the interface assertOwnership
// expects, decoupling the helper from the auth proto type.
type authUser struct {
	id   string
	role string
}

func (a *authUser) GetId() string   { return a.id }
func (a *authUser) GetRole() string { return a.role }

// GetLimitOffset extracts limit and offset from query parameters with defaults
func GetLimitOffset(r *http.Request, defaultLimit, defaultOffset int) (limit, offset int) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 {
			limit = l
		} else {
			limit = defaultLimit
		}
	} else {
		limit = defaultLimit
	}

	if offsetStr != "" {
		o, err := strconv.Atoi(offsetStr)
		if err == nil && o >= 0 {
			offset = o
		} else {
			offset = defaultOffset
		}
	} else {
		offset = defaultOffset
	}

	return limit, offset
}
