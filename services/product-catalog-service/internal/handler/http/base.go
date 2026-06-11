package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	appErr "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/errors"
)

// Response is a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a JSON response
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// SuccessResponse returns a successful response
func SuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	response := Response{
		Success: true,
		Data:    data,
	}
	JSON(w, statusCode, response)
}

// ErrorResponse returns an error response
func ErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	response := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
	JSON(w, statusCode, response)
}

// HandleError handles application errors and returns appropriate HTTP status codes
func HandleError(w http.ResponseWriter, err error) {
	appError, ok := err.(*appErr.AppError)
	if !ok {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	switch appError.Type {
	case appErr.NotFound:
		ErrorResponse(w, http.StatusNotFound, string(appError.Code), appError.Message)
	case appErr.Conflict:
		ErrorResponse(w, http.StatusConflict, string(appError.Code), appError.Message)
	case appErr.Validation:
		ErrorResponse(w, http.StatusBadRequest, string(appError.Code), appError.Message)
	case appErr.Unauthorized:
		ErrorResponse(w, http.StatusUnauthorized, string(appError.Code), appError.Message)
	default:
		ErrorResponse(w, http.StatusInternalServerError, string(appError.Code), appError.Message)
	}
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
