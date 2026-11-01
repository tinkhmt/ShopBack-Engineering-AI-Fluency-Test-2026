package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is loaded once at startup. Nothing else in the app reads os.Getenv.
type Config struct {
	Env          string
	HTTPAddr     string
	DatabaseURL  string
	JWTSecret    string
	JWTTTL       time.Duration
	CookieSecure bool
}

// Load reads the environment and returns every problem at once, so a
// misconfigured deployment fails on the first run instead of on first use.
func Load() (*Config, error) {
	cfg := &Config{
		Env:         getenv("APP_ENV", "development"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTTTL:      24 * time.Hour,
	}

	var problems []string

	if cfg.DatabaseURL == "" {
		problems = append(problems, "DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		problems = append(problems, "JWT_SECRET is required and must be at least 32 characters")
	}

	if raw := os.Getenv("JWT_TTL"); raw != "" {
		ttl, err := time.ParseDuration(raw)
		if err != nil || ttl <= 0 {
			problems = append(problems, "JWT_TTL must be a positive duration such as 24h")
		} else {
			cfg.JWTTTL = ttl
		}
	}

	if raw := os.Getenv("COOKIE_SECURE"); raw != "" {
		secure, err := strconv.ParseBool(raw)
		if err != nil {
			problems = append(problems, "COOKIE_SECURE must be true or false")
		} else {
			cfg.CookieSecure = secure
		}
	}

	if cfg.IsProduction() && !cfg.CookieSecure {
		problems = append(problems, "COOKIE_SECURE must be true when APP_ENV=production")
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}

	return cfg, nil
}

func (c *Config) IsProduction() bool { return c.Env == "production" }

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
