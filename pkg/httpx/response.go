package httpx

import (
	"encoding/json"
	"net/http"
)

type envelope struct {
	Success bool   `json:"success"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// JSON writes v as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Success writes a standard success envelope.
func Success(w http.ResponseWriter, status int, data any) {
	JSON(w, status, envelope{Success: true, Data: data})
}

// Error writes a standard error envelope.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, envelope{Success: false, Code: code, Message: message})
}

// pageEnvelope is the standard shape for any paginated list response, shared
// across services so clients only need to handle one list-response pattern.
type pageEnvelope[T any] struct {
	Success  bool  `json:"success"`
	Data     []T   `json:"data"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// Paginated writes a standard paginated list envelope:
//
//	{ "success": true, "data": [...], "total": N, "page": N, "page_size": N }
//
// page is 1-indexed. data is never null (callers should pass a non-nil,
// possibly-empty slice).
func Paginated[T any](w http.ResponseWriter, status int, data []T, total int64, page, pageSize int) {
	if data == nil {
		data = []T{}
	}
	JSON(w, status, pageEnvelope[T]{
		Success:  true,
		Data:     data,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
