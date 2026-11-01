package config

import (
	"strings"
	"testing"
	"time"
)

func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("JWT_TTL", "")
	t.Setenv("COOKIE_SECURE", "")
}

func TestLoadDefaults(t *testing.T) {
	setBaseEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != "development" {
		t.Errorf("Env = %q, want development", cfg.Env)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.JWTTTL != 30*time.Minute {
		t.Errorf("JWTTTL = %v, want 30m", cfg.JWTTTL)
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure = true, want false in development")
	}
}

func TestLoadReportsAllMissingRequiredValues(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	for _, want := range []string{"DATABASE_URL", "JWT_SECRET"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
}

func TestLoadRejectsShortSecret(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("JWT_SECRET", "too-short")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error for short JWT_SECRET")
	}
}

func TestLoadRejectsInvalidOptionalValues(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{name: "ttl not a duration", key: "JWT_TTL", val: "soon"},
		{name: "ttl not positive", key: "JWT_TTL", val: "-1h"},
		{name: "cookie secure not a bool", key: "COOKIE_SECURE", val: "maybe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setBaseEnv(t)
			t.Setenv(tt.key, tt.val)

			if _, err := Load(); err == nil {
				t.Fatalf("Load() error = nil, want error for %s=%q", tt.key, tt.val)
			}
		})
	}
}

func TestProductionRequiresSecureCookie(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want error for insecure production cookie")
	}

	t.Setenv("COOKIE_SECURE", "true")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.IsProduction() {
		t.Error("IsProduction() = false, want true")
	}
}
