package middleware

import (
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/tinkhmt/shopback-todo/backend/internal/errors"
)

// maxBodyBytes caps request bodies. The largest legitimate payload is a task
// description of 2000 runes (~8 KiB), so 64 KiB leaves ample headroom.
const maxBodyBytes = 64 << 10

// Security sets response headers that remove whole classes of browser-side
// attacks at no cost.
func Security(cookieSecure bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cache-Control", "no-store")
		if cookieSecure {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

// BodyLimit rejects oversized bodies while they are read, so a client cannot
// force the server to buffer an unbounded payload. The read error is mapped to
// 413 by errors.BindError.
func BodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		c.Next()
	}
}

// RequireJSON enforces the "JSON-only bodies" CSRF mitigation: a cross-site
// HTML form can only send urlencoded, multipart or text/plain, all rejected.
func RequireJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodDelete:
			c.Next()
			return
		}
		if c.Request.ContentLength == 0 {
			c.Next()
			return
		}

		mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if err != nil || mediaType != "application/json" {
			apperrors.Fail(c, apperrors.UnsupportedMediaType("content type must be application/json"))
			return
		}
		c.Next()
	}
}
