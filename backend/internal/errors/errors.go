package errors

import "net/http"

// AppError is an error that is safe to translate into an HTTP response.
// The service layer returns these; handlers never pick status codes themselves.
type AppError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
	cause   error
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.cause }

func BadRequest(code, message string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: code, Message: message}
}

func Validation(fields map[string]string) *AppError {
	return &AppError{
		Status:  http.StatusBadRequest,
		Code:    "validation_failed",
		Message: "some fields are invalid",
		Fields:  fields,
	}
}

func Unauthorized(message string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: message}
}

func NotFound(message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "not_found", Message: message}
}

func MethodNotAllowed() *AppError {
	return &AppError{Status: http.StatusMethodNotAllowed, Code: "method_not_allowed", Message: "method not allowed"}
}

func Conflict(code, message string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: code, Message: message}
}

func ServiceUnavailable(message string) *AppError {
	return &AppError{Status: http.StatusServiceUnavailable, Code: "service_unavailable", Message: message}
}

// Internal wraps an unexpected cause. The cause is logged, never sent to the client.
func Internal(cause error) *AppError {
	return &AppError{
		Status:  http.StatusInternalServerError,
		Code:    "internal_error",
		Message: "internal server error",
		cause:   cause,
	}
}
