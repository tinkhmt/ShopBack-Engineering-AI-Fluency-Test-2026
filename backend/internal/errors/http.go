package errors

import (
	stderrors "errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

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
	var appErr *AppError
	if !stderrors.As(err, &appErr) {
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

// BindError turns Gin/validator binding failures into a field-level 400.
func BindError(err error) *AppError {
	// MaxBytesReader aborts mid-read; without this the client would see a
	// misleading 400 instead of 413.
	var maxBytesErr *http.MaxBytesError
	if stderrors.As(err, &maxBytesErr) {
		return PayloadTooLarge("request body is too large")
	}

	var validationErrs validator.ValidationErrors
	if !stderrors.As(err, &validationErrs) {
		return BadRequest("invalid_request", "request body must be valid JSON")
	}

	fields := make(map[string]string, len(validationErrs))
	for _, fieldErr := range validationErrs {
		fields[lowerFirst(fieldErr.Field())] = validationMessage(fieldErr)
	}
	return Validation(fields)
}

func validationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + fieldErr.Param() + " characters"
	case "max":
		return "must be at most " + fieldErr.Param() + " characters"
	default:
		return "is invalid"
	}
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	return string(value[0]+('a'-'A')) + value[1:]
}
