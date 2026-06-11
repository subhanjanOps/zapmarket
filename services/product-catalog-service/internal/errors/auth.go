package errors

func UserNotFound() error {
	return &AppError{
		Type:    NotFound,
		Code:    "USER_NOT_FOUND",
		Message: "user not found",
	}
}

func InvalidPassword() error {
	return &AppError{
		Type:    Unauthorized,
		Code:    "INVALID_PASSWORD",
		Message: "invalid password",
	}
}

func UserAlreadyExists() error {
	return &AppError{
		Type:    Conflict,
		Code:    "USER_ALREADY_EXISTS",
		Message: "user already exists",
	}
}
