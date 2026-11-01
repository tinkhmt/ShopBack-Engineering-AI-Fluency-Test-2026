package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/tinkhmt/shopback-todo/backend/internal/config"
	"github.com/tinkhmt/shopback-todo/backend/internal/util"
)

func newSecureTestRouter() *gin.Engine {
	return buildRouter(&config.Config{Env: "test", JWTTTL: time.Minute, CookieSecure: true},
		fakePinger{}, newMemoryTaskRepository())
}

func TestSecurityHeaders(t *testing.T) {
	w := perform(newTestRouter(fakePinger{}), http.MethodGet, "/healthz", "", nil)

	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		"Cache-Control":           "no-store",
	}
	for header, value := range want {
		if got := w.Header().Get(header); got != value {
			t.Errorf("%s = %q, want %q", header, got, value)
		}
	}
	if got := w.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("HSTS = %q over plain HTTP, want empty", got)
	}
}

func TestHSTSWhenCookieSecure(t *testing.T) {
	w := perform(newSecureTestRouter(), http.MethodGet, "/healthz", "", nil)

	if got := w.Header().Get("Strict-Transport-Security"); !strings.Contains(got, "max-age=31536000") {
		t.Errorf("HSTS = %q, want max-age=31536000", got)
	}
}

func TestSessionRejectsExpiredToken(t *testing.T) {
	expired := util.NewTokenManager(strings.Repeat("s", 32), -time.Minute)
	raw, err := expired.Issue(1)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	assertUnauthorized(t, raw)
}

func TestSessionRejectsTamperedToken(t *testing.T) {
	tokens := util.NewTokenManager(strings.Repeat("s", 32), time.Minute)
	raw, err := tokens.Issue(1)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	assertUnauthorized(t, raw+"x")
}

func TestSessionRejectsNoneAlgorithm(t *testing.T) {
	claims := jwt.RegisteredClaims{
		Subject:   "1",
		Issuer:    "shopback-todo",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("building unsigned token: %v", err)
	}

	assertUnauthorized(t, raw)
}

func assertUnauthorized(t *testing.T, token string) {
	t.Helper()
	cookie := &http.Cookie{Name: util.SessionCookieName, Value: token}

	w := perform(newTestRouter(fakePinger{}), http.MethodGet, "/api/auth/me", "", cookie)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	body := decodeError(t, w)
	if body.Error.Code != "unauthorized" {
		t.Errorf("code = %q, want unauthorized", body.Error.Code)
	}
	if body.Error.Message != "session is invalid or expired" {
		t.Errorf("message = %q", body.Error.Message)
	}
}

func TestOversizedBodyIsRejected(t *testing.T) {
	body := `{"email":"` + strings.Repeat("a", 70<<10) + `","password":"password123"}`

	w := perform(newTestRouter(fakePinger{}), http.MethodPost, "/api/auth/register", body, nil)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", w.Code)
	}
	if code := decodeError(t, w).Error.Code; code != "payload_too_large" {
		t.Errorf("code = %q, want payload_too_large", code)
	}
}

func TestNonJSONBodyIsRejected(t *testing.T) {
	for _, contentType := range []string{"text/plain", "application/x-www-form-urlencoded", "multipart/form-data"} {
		t.Run(contentType, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/auth/register",
				strings.NewReader(`{"email":"alice@example.com","password":"password123"}`))
			request.Header.Set("Content-Type", contentType)

			w := httptest.NewRecorder()
			newTestRouter(fakePinger{}).ServeHTTP(w, request)

			if w.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("status = %d, want 415", w.Code)
			}
			if code := decodeError(t, w).Error.Code; code != "unsupported_media_type" {
				t.Errorf("code = %q, want unsupported_media_type", code)
			}
		})
	}
}

func TestJSONContentTypeWithCharsetIsAccepted(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/register",
		strings.NewReader(`{"email":"alice@example.com","password":"password123"}`))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	w := httptest.NewRecorder()
	newTestRouter(fakePinger{}).ServeHTTP(w, request)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
}
