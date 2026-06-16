package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
	"github.com/zapmarket/zapmarket/pkg/httpx"
)

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
