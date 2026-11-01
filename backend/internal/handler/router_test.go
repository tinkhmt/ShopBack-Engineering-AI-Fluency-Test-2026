package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tinkhmt/shopback-todo/backend/internal/config"
	"github.com/tinkhmt/shopback-todo/backend/internal/handler"
	"github.com/tinkhmt/shopback-todo/backend/internal/model"
	"github.com/tinkhmt/shopback-todo/backend/internal/repository"
	"github.com/tinkhmt/shopback-todo/backend/internal/service"
	"github.com/tinkhmt/shopback-todo/backend/internal/util"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

type memoryRepository struct {
	byEmail map[string]*model.User
	nextID  int64
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{byEmail: make(map[string]*model.User)}
}

func (m *memoryRepository) Create(_ context.Context, email, passwordHash string) (*model.User, error) {
	if _, exists := m.byEmail[email]; exists {
		return nil, repository.ErrEmailTaken
	}
	m.nextID++
	user := &model.User{ID: m.nextID, Email: email, PasswordHash: passwordHash, CreatedAt: time.Now()}
	m.byEmail[email] = user
	return user, nil
}

func (m *memoryRepository) FindByEmail(_ context.Context, email string) (*model.User, error) {
	user, ok := m.byEmail[email]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return user, nil
}

func (m *memoryRepository) FindByID(_ context.Context, id int64) (*model.User, error) {
	for _, user := range m.byEmail {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func newTestRouter(db handler.Pinger) *gin.Engine {
	tokens := util.NewTokenManager(strings.Repeat("s", 32), time.Minute)
	svc := service.NewAuthService(newMemoryRepository(), tokens)
	authHandler := handler.NewAuthHandler(svc, false, time.Minute)
	return handler.NewRouter(&config.Config{Env: "test", JWTTTL: time.Minute}, db, authHandler, tokens)
}

type errorEnvelope struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	} `json:"error"`
}

type userEnvelope struct {
	User struct {
		ID        int64     `json:"id"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"createdAt"`
	} `json:"user"`
}

func perform(r *gin.Engine, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	var request *http.Request
	if body != "" {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	} else {
		request = httptest.NewRequest(method, path, nil)
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)
	return w
}

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), target); err != nil {
		t.Fatalf("response is not JSON: %v (body %q)", err, w.Body.String())
	}
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) errorEnvelope {
	t.Helper()
	var body errorEnvelope
	decodeJSON(t, w, &body)
	return body
}

func sessionCookie(w *httptest.ResponseRecorder) *http.Cookie {
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == util.SessionCookieName {
			return cookie
		}
	}
	return nil
}

func registerUser(t *testing.T, r *gin.Engine) *http.Cookie {
	t.Helper()
	w := perform(r, http.MethodPost, "/api/auth/register",
		`{"email":"alice@example.com","password":"password123"}`, nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body %s", w.Code, w.Body.String())
	}
	cookie := sessionCookie(w)
	if cookie == nil {
		t.Fatal("register did not set a session cookie")
	}
	return cookie
}

// --- router: health, 404, 405, panic ---

func TestHealthzOK(t *testing.T) {
	w := perform(newTestRouter(fakePinger{}), http.MethodGet, "/healthz", "", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if w.Body.String() != `{"status":"ok"}` {
		t.Errorf("body = %s", w.Body.String())
	}
}

func TestHealthzDatabaseDown(t *testing.T) {
	w := perform(newTestRouter(fakePinger{err: errors.New("connection refused")}),
		http.MethodGet, "/healthz", "", nil)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if code := decodeError(t, w).Error.Code; code != "service_unavailable" {
		t.Errorf("code = %q, want service_unavailable", code)
	}
}

func TestUnknownRouteUsesErrorEnvelope(t *testing.T) {
	w := perform(newTestRouter(fakePinger{}), http.MethodGet, "/does-not-exist", "", nil)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	if code := decodeError(t, w).Error.Code; code != "not_found" {
		t.Errorf("code = %q, want not_found", code)
	}
}

func TestWrongMethodUsesErrorEnvelope(t *testing.T) {
	w := perform(newTestRouter(fakePinger{}), http.MethodPost, "/healthz", "", nil)

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

	w := perform(r, http.MethodGet, "/boom", "", nil)

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

// --- auth handlers ---

func TestRegisterSetsHardenedSessionCookie(t *testing.T) {
	w := perform(newTestRouter(fakePinger{}), http.MethodPost, "/api/auth/register",
		`{"email":"  Alice@Example.COM  ","password":"password123"}`, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}

	cookie := sessionCookie(w)
	if cookie == nil {
		t.Fatal("no session cookie")
	}
	if !cookie.HttpOnly {
		t.Error("cookie is not HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", cookie.SameSite)
	}
	if cookie.Path != "/" {
		t.Errorf("Path = %q, want /", cookie.Path)
	}
	if cookie.MaxAge != 60 {
		t.Errorf("MaxAge = %d, want 60 (1 minute TTL)", cookie.MaxAge)
	}
	if cookie.Value == "" {
		t.Error("cookie value is empty")
	}

	var body userEnvelope
	decodeJSON(t, w, &body)
	if body.User.Email != "alice@example.com" {
		t.Errorf("email = %q, want normalized alice@example.com", body.User.Email)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	r := newTestRouter(fakePinger{})
	registerUser(t, r)

	w := perform(r, http.MethodPost, "/api/auth/register",
		`{"email":"ALICE@example.com","password":"password123"}`, nil)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
	if code := decodeError(t, w).Error.Code; code != "email_taken" {
		t.Errorf("code = %q, want email_taken", code)
	}
}

func TestRegisterValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		field string
	}{
		{
			name:  "invalid email is caught by the service",
			body:  `{"email":"not-an-email","password":"password123"}`,
			field: "email",
		},
		{
			name:  "short password is caught by binding",
			body:  `{"email":"alice@example.com","password":"short"}`,
			field: "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := perform(newTestRouter(fakePinger{}), http.MethodPost, "/api/auth/register", tt.body, nil)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
			body := decodeError(t, w)
			if body.Error.Code != "validation_failed" {
				t.Errorf("code = %q", body.Error.Code)
			}
			if body.Error.Fields[tt.field] == "" {
				t.Errorf("missing validation message for %q (fields %v)", tt.field, body.Error.Fields)
			}
		})
	}
}

func TestLoginSetsSessionCookie(t *testing.T) {
	r := newTestRouter(fakePinger{})
	registerUser(t, r)

	w := perform(r, http.MethodPost, "/api/auth/login",
		`{"email":"ALICE@example.com","password":"password123"}`, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	if sessionCookie(w) == nil {
		t.Fatal("login did not set a session cookie")
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	r := newTestRouter(fakePinger{})
	registerUser(t, r)

	w := perform(r, http.MethodPost, "/api/auth/login",
		`{"email":"alice@example.com","password":"wrong-password"}`, nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if message := decodeError(t, w).Error.Message; message != "invalid email or password" {
		t.Errorf("message = %q", message)
	}
}

func TestMeRequiresAuthentication(t *testing.T) {
	w := perform(newTestRouter(fakePinger{}), http.MethodGet, "/api/auth/me", "", nil)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestMeReturnsCurrentUser(t *testing.T) {
	r := newTestRouter(fakePinger{})
	cookie := registerUser(t, r)

	w := perform(r, http.MethodGet, "/api/auth/me", "", cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var body userEnvelope
	decodeJSON(t, w, &body)
	if body.User.Email != "alice@example.com" {
		t.Errorf("email = %q", body.User.Email)
	}
}

func TestMeRejectsForgedCookie(t *testing.T) {
	forged := &http.Cookie{Name: util.SessionCookieName, Value: "not-a-token"}

	w := perform(newTestRouter(fakePinger{}), http.MethodGet, "/api/auth/me", "", forged)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestLogoutClearsCookie(t *testing.T) {
	r := newTestRouter(fakePinger{})
	cookie := registerUser(t, r)

	w := perform(r, http.MethodPost, "/api/auth/logout", "", cookie)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}

	cleared := sessionCookie(w)
	if cleared == nil || cleared.Value != "" {
		t.Fatalf("cleared cookie = %+v, want empty value", cleared)
	}
	if header := w.Header().Get("Set-Cookie"); !strings.Contains(header, "Max-Age=0") {
		t.Errorf("Set-Cookie = %q, want Max-Age=0", header)
	}
}

func TestMalformedJSON(t *testing.T) {
	w := perform(newTestRouter(fakePinger{}), http.MethodPost, "/api/auth/register", `{"email":`, nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if code := decodeError(t, w).Error.Code; code != "invalid_request" {
		t.Errorf("code = %q, want invalid_request", code)
	}
}
