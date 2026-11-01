package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tinkhmt/shopback-todo/backend/internal/config"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newTestRouter(db Pinger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return New(&config.Config{Env: "test"}, db)
}

func do(t *testing.T, r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) errorEnvelope {
	t.Helper()
	var body errorEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v (body %q)", err, w.Body.String())
	}
	return body
}

func TestHealthzOK(t *testing.T) {
	w := do(t, newTestRouter(fakePinger{}), http.MethodGet, "/healthz")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if w.Body.String() != `{"status":"ok"}` {
		t.Errorf("body = %s", w.Body.String())
	}
}

func TestHealthzDatabaseDown(t *testing.T) {
	w := do(t, newTestRouter(fakePinger{err: errors.New("connection refused")}), http.MethodGet, "/healthz")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if code := decodeError(t, w).Error.Code; code != "service_unavailable" {
		t.Errorf("code = %q, want service_unavailable", code)
	}
}

func TestUnknownRouteUsesErrorEnvelope(t *testing.T) {
	w := do(t, newTestRouter(fakePinger{}), http.MethodGet, "/does-not-exist")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if code := decodeError(t, w).Error.Code; code != "not_found" {
		t.Errorf("code = %q, want not_found", code)
	}
}

func TestWrongMethodUsesErrorEnvelope(t *testing.T) {
	w := do(t, newTestRouter(fakePinger{}), http.MethodPost, "/healthz")

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
	if code := decodeError(t, w).Error.Code; code != "method_not_allowed" {
		t.Errorf("code = %q, want method_not_allowed", code)
	}
}

func TestPanicIsRecoveredAsJSON(t *testing.T) {
	r := newTestRouter(fakePinger{})
	r.GET("/boom", func(*gin.Context) { panic("boom") })

	w := do(t, r, http.MethodGet, "/boom")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	body := decodeError(t, w)
	if body.Error.Code != "internal_error" {
		t.Errorf("code = %q, want internal_error", body.Error.Code)
	}
	if body.Error.Message != "internal server error" {
		t.Errorf("message = %q leaks internals", body.Error.Message)
	}
}
