package errors

type ErrorType string

const (
	NotFound     ErrorType = "not_found"
	Conflict     ErrorType = "conflict"
	Internal     ErrorType = "internal"
	Validation   ErrorType = "validation"
	Unauthorized ErrorType = "unauthorized"
)

type ErrorCode string

const (
	DATABASE_ERROR   ErrorCode = "DATABASE_ERROR"
	CONFLICT_ERROR   ErrorCode = "ALREADY_EXISTS"
	NOT_FOUND_ERROR  ErrorCode = "NOT_FOUND"
	VALIDATION_ERROR ErrorCode = "INVALID_DATA"
	INTERNAL_ERROR   ErrorCode = "INTERNAL_SERVER_ERROR"
)

type AppError struct {
	Type    ErrorType
	Code    ErrorCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func DatabaseError(message string, err error) error {
	return &AppError{
		Type:    Internal,
		Code:    DATABASE_ERROR,
		Message: message,
		Err:     err,
	}
}

func ConflictError(message string, err error) error {
	return &AppError{
		Type:    Conflict,
		Code:    CONFLICT_ERROR,
		Message: message,
		Err:     err,
	}
}

func NotFoundError(message string, err error) error {
	return &AppError{
		Type:    NotFound,
		Code:    NOT_FOUND_ERROR,
		Message: message,
		Err:     err,
	}
}

func ValidationError(message string, err error) error {
	return &AppError{
		Type:    Validation,
		Code:    VALIDATION_ERROR,
		Message: message,
		Err:     err,
	}
}
func InternalError(message string, err error) error {
	return &AppError{
		Type:    Internal,
		Code:    INTERNAL_ERROR,
		Message: message,
		Err:     err,
	}
}
