package server

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tinkhmt/shopback-todo/backend/internal/config"
	"github.com/tinkhmt/shopback-todo/backend/internal/httpx"
	"github.com/tinkhmt/shopback-todo/backend/internal/middleware"
)

// Pinger is the only database capability the health check needs.
type Pinger interface {
	Ping(ctx context.Context) error
}

// New builds the router. Modules are mounted here as they are added.
func New(cfg *config.Config, db Pinger) *gin.Engine {
	// Release mode suppresses Gin's route dump; the structured logger already
	// reports every request.
	if cfg.Env == "test" {
		gin.SetMode(gin.TestMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// No reverse proxy in front of this app; trusting X-Forwarded-For would
	// let clients spoof c.ClientIP().
	_ = r.SetTrustedProxies(nil)

	r.Use(middleware.Recovery(), middleware.Logger())

	r.GET("/healthz", health(db))

	r.HandleMethodNotAllowed = true
	r.NoRoute(func(c *gin.Context) {
		httpx.Fail(c, httpx.NotFound("route not found"))
	})
	r.NoMethod(func(c *gin.Context) {
		httpx.Fail(c, httpx.MethodNotAllowed())
	})

	return r
}

func health(db Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			httpx.Fail(c, httpx.ServiceUnavailable("database unavailable"))
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	}
}
