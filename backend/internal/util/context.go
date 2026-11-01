package util

import "github.com/gin-gonic/gin"

const userIDKey = "auth.user_id"

// SetUserID stores the authenticated user for downstream handlers.
func SetUserID(c *gin.Context, id int64) {
	c.Set(userIDKey, id)
}

// UserID reads the authenticated user without an unchecked type assertion.
func UserID(c *gin.Context) (int64, bool) {
	value, ok := c.Get(userIDKey)
	if !ok {
		return 0, false
	}
	id, ok := value.(int64)
	return id, ok
}
