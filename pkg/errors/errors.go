package errors

import (
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorType classifies the kind of error for HTTP/gRPC mapping.
type ErrorType string

const (
	NotFound     ErrorType = "not_found"
	Conflict     ErrorType = "conflict"
	Internal     ErrorType = "internal"
	Validation   ErrorType = "validation"
	Unauthorized ErrorType = "unauthorized"
)

// AppError is the standard application error shared across all services.
type AppError struct {
	Type    ErrorType
	Code    string // uppercase string code, e.g. "USER_NOT_FOUND"
	Message string // human-readable message
	Err     error  // wrapped underlying error
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

// --- Constructors ---

func NewNotFound(code, message string) *AppError {
	return &AppError{Type: NotFound, Code: code, Message: message}
}

func NewConflict(code, message string) *AppError {
	return &AppError{Type: Conflict, Code: code, Message: message}
}

func NewValidation(code, message string) *AppError {
	return &AppError{Type: Validation, Code: code, Message: message}
}

func NewUnauthorized(code, message string) *AppError {
	return &AppError{Type: Unauthorized, Code: code, Message: message}
}

func NewInternal(code, message string, err error) *AppError {
	return &AppError{Type: Internal, Code: code, Message: message, Err: err}
}

// --- HTTP mapping ---

// HTTPStatus returns the HTTP status code for an AppError type.
func (e *AppError) HTTPStatus() int {
	switch e.Type {
	case NotFound:
		return http.StatusNotFound
	case Conflict:
		return http.StatusConflict
	case Validation:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// --- HTTP mapping ---

// HandleHTTP writes a standard JSON error response for err.
// Uses the same envelope as pkg/httpx: {"success":false,"code":"...","message":"..."}.
func HandleHTTP(w http.ResponseWriter, err error) {
	appErr, ok := err.(*AppError)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}
	writeJSON(w, appErr.HTTPStatus(), appErr.Code, appErr.Message)
}

func writeJSON(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}{false, code, message})
}

// --- gRPC mapping ---

// ToGRPCStatus converts err to a gRPC status error.
// If err is not an *AppError, codes.Internal is used.
func ToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}
	appErr, ok := err.(*AppError)
	if !ok {
		return status.Error(codes.Internal, "internal server error")
	}
	switch appErr.Type {
	case NotFound:
		return status.Error(codes.NotFound, appErr.Message)
	case Conflict:
		return status.Error(codes.AlreadyExists, appErr.Message)
	case Validation:
		return status.Error(codes.InvalidArgument, appErr.Message)
	case Unauthorized:
		return status.Error(codes.PermissionDenied, appErr.Message)
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
