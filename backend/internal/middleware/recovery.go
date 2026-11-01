package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	apperrors "github.com/tinkhmt/shopback-todo/backend/internal/errors"
)

// Recovery converts a panic into the same JSON error envelope as every other
// failure, and logs the stack trace server-side.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		slog.Error("panic recovered",
			"error", fmt.Sprint(recovered),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"stack", string(debug.Stack()),
		)
		apperrors.Fail(c, apperrors.Internal(fmt.Errorf("panic: %v", recovered)))
	})
}
