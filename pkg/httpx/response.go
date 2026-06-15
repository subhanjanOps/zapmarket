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
