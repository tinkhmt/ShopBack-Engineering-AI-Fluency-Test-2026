package logger

import (
	"log/slog"
	"os"

	"github.com/tinkhmt/shopback-todo/backend/internal/config"
)

// New builds the process logger: JSON in production for log collectors, text
// elsewhere for readability.
func New(cfg *config.Config) *slog.Logger {
	if cfg.IsProduction() {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}
