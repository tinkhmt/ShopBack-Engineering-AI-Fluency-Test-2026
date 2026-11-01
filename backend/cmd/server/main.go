package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tinkhmt/shopback-todo/backend/internal/config"
	"github.com/tinkhmt/shopback-todo/backend/internal/database"
	"github.com/tinkhmt/shopback-todo/backend/internal/handler"
	"github.com/tinkhmt/shopback-todo/backend/internal/logger"
	"github.com/tinkhmt/shopback-todo/backend/internal/repository"
	"github.com/tinkhmt/shopback-todo/backend/internal/service"
	"github.com/tinkhmt/shopback-todo/backend/internal/util"
)

func main() {
	if err := run(); err != nil {
		// Plain output: config errors are multi-line and the logger may not be
		// configured yet.
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	slog.SetDefault(logger.New(cfg))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Composition root: every layer receives its dependencies explicitly.
	userRepository := repository.NewUserRepository(pool)
	taskRepository := repository.NewTaskRepository(pool)
	tokenManager := util.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	authService := service.NewAuthService(userRepository, tokenManager)
	taskService := service.NewTaskService(taskRepository)
	authHandler := handler.NewAuthHandler(authService, cfg.CookieSecure, cfg.JWTTTL)
	taskHandler := handler.NewTaskHandler(taskService)

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: handler.NewRouter(cfg, handler.Deps{
			DB:     pool,
			Auth:   authHandler,
			Tasks:  taskHandler,
			Tokens: tokenManager,
		}),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	listenErr := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", cfg.HTTPAddr, "env", cfg.Env)
		listenErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-listenErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	slog.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	slog.Info("server stopped")
	return nil
}
