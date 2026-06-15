package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	pkgerrors "github.com/zapmarket/zapmarket/pkg/errors"
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
	JSON(w, statusCode, Response{Success: true, Data: data})
}

// ErrorResponse returns an error response
func ErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	JSON(w, statusCode, Response{
		Success: false,
		Error:   &ErrorInfo{Code: code, Message: message},
	})
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
