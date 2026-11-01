package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error is an error that is safe to translate into an HTTP response.
// The service layer returns these; handlers never pick status codes themselves.
type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
	cause   error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.cause }

func BadRequest(code, message string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: code, Message: message}
}

func Validation(fields map[string]string) *Error {
	return &Error{
		Status:  http.StatusBadRequest,
		Code:    "validation_failed",
		Message: "some fields are invalid",
		Fields:  fields,
	}
}

func Unauthorized(message string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: message}
}

func NotFound(message string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: message}
}

func MethodNotAllowed() *Error {
	return &Error{Status: http.StatusMethodNotAllowed, Code: "method_not_allowed", Message: "method not allowed"}
}

func Conflict(code, message string) *Error {
	return &Error{Status: http.StatusConflict, Code: code, Message: message}
}

func ServiceUnavailable(message string) *Error {
	return &Error{Status: http.StatusServiceUnavailable, Code: "service_unavailable", Message: message}
}

// Internal wraps an unexpected cause. The cause is logged, never sent to the client.
func Internal(cause error) *Error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Code:    "internal_error",
		Message: "internal server error",
		cause:   cause,
	}
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// Fail is the single place that turns an error into an HTTP response.
func Fail(c *gin.Context, err error) {
	var appErr *Error
	if !errors.As(err, &appErr) {
		appErr = Internal(err)
	}

	if appErr.Status >= http.StatusInternalServerError {
		slog.ErrorContext(c.Request.Context(), "request failed",
			"error", appErr.Unwrap(),
			"code", appErr.Code,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
		)
	}

	c.AbortWithStatusJSON(appErr.Status, errorBody{Error: errorDetail{
		Code:    appErr.Code,
		Message: appErr.Message,
		Fields:  appErr.Fields,
	}})
}
