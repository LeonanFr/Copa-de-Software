package shared

import "net/http"

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func NewBadRequestError(message string, err error) *AppError {
	return &AppError{Code: 400, Message: message, Err: err}
}

func NewNotFoundError(message string, err error) *AppError {
	return &AppError{Code: 404, Message: message, Err: err}
}

func NewConflictError(message string, err error) *AppError {
	return &AppError{Code: 409, Message: message, Err: err}
}

func NewInternalServerError(message string, err error) *AppError {
	return &AppError{Code: 500, Message: message, Err: err}
}

func NewUnauthorizedError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusUnauthorized,
		Message: message,
		Err:     err,
	}
}
