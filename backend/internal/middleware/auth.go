package middleware

import (
	"github.com/gin-gonic/gin"

	apperrors "github.com/tinkhmt/shopback-todo/backend/internal/errors"
	"github.com/tinkhmt/shopback-todo/backend/internal/util"
)

// RequireSession rejects requests without a valid session cookie and exposes
// the user ID to downstream handlers through util.UserID.
func RequireSession(tokens *util.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(util.SessionCookieName)
		if err != nil || raw == "" {
			apperrors.Fail(c, apperrors.Unauthorized("authentication required"))
			return
		}

		userID, err := tokens.Parse(raw)
		if err != nil {
			apperrors.Fail(c, apperrors.Unauthorized("session is invalid or expired"))
			return
		}

		util.SetUserID(c, userID)
		c.Next()
	}
}
