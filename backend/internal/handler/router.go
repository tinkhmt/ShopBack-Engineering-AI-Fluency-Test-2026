package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tinkhmt/shopback-todo/backend/internal/config"
	apperrors "github.com/tinkhmt/shopback-todo/backend/internal/errors"
	"github.com/tinkhmt/shopback-todo/backend/internal/middleware"
	"github.com/tinkhmt/shopback-todo/backend/internal/util"
)

// Pinger is the only database capability the health check needs.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Deps are the already-built handlers and shared services the router mounts.
type Deps struct {
	DB     Pinger
	Auth   *AuthHandler
	Tasks  *TaskHandler
	Tokens *util.TokenManager
}

// NewRouter assembles the HTTP surface. It receives already-built handlers, so
// wiring stays in cmd/server/main.go.
func NewRouter(cfg *config.Config, deps Deps) *gin.Engine {
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

	r.Use(middleware.Recovery(), middleware.Logger(), middleware.Security(cfg.CookieSecure),
		middleware.BodyLimit(), middleware.RequireJSON())

	r.GET("/healthz", health(deps.DB))

	authRoutes := r.Group("/api/auth")
	authRoutes.POST("/register", deps.Auth.Register)
	authRoutes.POST("/login", deps.Auth.Login)
	authRoutes.POST("/logout", deps.Auth.Logout)
	authRoutes.GET("/me", middleware.RequireSession(deps.Tokens), deps.Auth.Me)

	taskRoutes := r.Group("/api/tasks", middleware.RequireSession(deps.Tokens))
	taskRoutes.GET("", deps.Tasks.List)
	taskRoutes.POST("", deps.Tasks.Create)
	taskRoutes.PATCH("/:id", deps.Tasks.Update)
	taskRoutes.DELETE("/:id", deps.Tasks.Delete)

	r.HandleMethodNotAllowed = true
	r.NoRoute(func(c *gin.Context) {
		apperrors.Fail(c, apperrors.NotFound("route not found"))
	})
	r.NoMethod(func(c *gin.Context) {
		apperrors.Fail(c, apperrors.MethodNotAllowed())
	})

	return r
}

func health(db Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			apperrors.Fail(c, apperrors.ServiceUnavailable("database unavailable"))
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
