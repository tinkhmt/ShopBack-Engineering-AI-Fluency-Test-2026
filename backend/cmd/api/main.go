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
	"github.com/tinkhmt/shopback-todo/backend/internal/server"
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

	slog.SetDefault(slog.New(newLogHandler(cfg)))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.New(cfg, pool),
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

func newLogHandler(cfg *config.Config) slog.Handler {
	if cfg.IsProduction() {
		return slog.NewJSONHandler(os.Stdout, nil)
	}
	return slog.NewTextHandler(os.Stdout, nil)
}
