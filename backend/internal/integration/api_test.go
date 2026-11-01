package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tinkhmt/shopback-todo/backend/internal/config"
	"github.com/tinkhmt/shopback-todo/backend/internal/handler"
	"github.com/tinkhmt/shopback-todo/backend/internal/repository"
	"github.com/tinkhmt/shopback-todo/backend/internal/service"
	"github.com/tinkhmt/shopback-todo/backend/internal/util"
)

const jwtSecret = "integration-test-secret-0123456789abcdef"

var pool *pgxpool.Pool

// TestMain runs the suite against a dedicated *_test database. Without
// TEST_DATABASE_URL the package skips, so plain `go test ./...` needs no Docker.
func TestMain(m *testing.M) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		fmt.Println("integration tests skipped: set TEST_DATABASE_URL (make test-integration)")
		os.Exit(0)
	}

	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse TEST_DATABASE_URL: %v\n", err)
		os.Exit(1)
	}
	// The schema reset below drops tables; never let a misconfigured URL point
	// it at a real database.
	if !strings.HasSuffix(cfg.ConnConfig.Database, "_test") {
		fmt.Fprintf(os.Stderr, "refusing to run: TEST_DATABASE_URL database %q does not end in _test\n",
			cfg.ConnConfig.Database)
		os.Exit(1)
	}

	pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to test database: %v\n", err)
		os.Exit(1)
	}
	if err := resetSchema(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "reset test schema: %v\n", err)
		pool.Close()
		os.Exit(1)
	}

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// resetSchema makes every run start from the same state. The .up.sql files are
// applied directly instead of through golang-migrate; switch when migrations
// need down/ordering semantics.
func resetSchema(ctx context.Context) error {
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS tasks, users CASCADE`); err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.up.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, file := range files {
		statement, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(statement)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(file), err)
		}
	}
	return nil
}

// newTestServer wires the real router exactly like cmd/server/main.go.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	tokens := util.NewTokenManager(jwtSecret, 30*time.Minute)
	authService := service.NewAuthService(repository.NewUserRepository(pool), tokens)
	taskService := service.NewTaskService(repository.NewTaskRepository(pool))

	router := handler.NewRouter(
		&config.Config{Env: "test", JWTTTL: 30 * time.Minute},
		handler.Deps{
			DB:     pool,
			Auth:   handler.NewAuthHandler(authService, false, 30*time.Minute),
			Tasks:  handler.NewTaskHandler(taskService),
			Tokens: tokens,
		},
	)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

// client is one browser: a cookie jar makes session handling real rather than
// hand-copied headers.
type client struct {
	t    *testing.T
	http *http.Client
	base string
}

func newClient(t *testing.T, server *httptest.Server) *client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	return &client{t: t, http: &http.Client{Jar: jar}, base: server.URL}
}

func (c *client) do(method, path string, body any) (int, []byte) {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("encode %s %s: %v", method, path, err)
		}
		reader = bytes.NewReader(raw)
	}

	request, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatalf("build %s %s: %v", method, path, err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		c.t.Fatalf("read %s %s: %v", method, path, err)
	}
	return response.StatusCode, payload
}

type taskPayload struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type userBody struct {
	User struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

type taskBody struct {
	Task taskPayload `json:"task"`
}

type tasksBody struct {
	Tasks []taskPayload `json:"tasks"`
}

type errorBody struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	} `json:"error"`
}

func decode[T any](t *testing.T, payload []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatalf("decode %q: %v", payload, err)
	}
	return value
}

func register(t *testing.T, c *client, email string) userBody {
	t.Helper()
	status, payload := c.do(http.MethodPost, "/api/auth/register",
		map[string]string{"email": email, "password": "password123"})
	if status != http.StatusCreated {
		t.Fatalf("register %s: status %d, body %s", email, status, payload)
	}
	return decode[userBody](t, payload)
}

func createTask(t *testing.T, c *client, title, description string) taskPayload {
	t.Helper()
	status, payload := c.do(http.MethodPost, "/api/tasks",
		map[string]string{"title": title, "description": description})
	if status != http.StatusCreated {
		t.Fatalf("create task %q: status %d, body %s", title, status, payload)
	}
	return decode[taskBody](t, payload).Task
}

func listTasks(t *testing.T, c *client, query string) []taskPayload {
	t.Helper()
	status, payload := c.do(http.MethodGet, "/api/tasks"+query, nil)
	if status != http.StatusOK {
		t.Fatalf("list tasks%s: status %d, body %s", query, status, payload)
	}
	return decode[tasksBody](t, payload).Tasks
}

func taskPath(id int64) string {
	return fmt.Sprintf("/api/tasks/%d", id)
}

func TestFullFlowTwoUsers(t *testing.T) {
	server := newTestServer(t)
	alice := newClient(t, server)
	bob := newClient(t, server)
	anonymous := newClient(t, server)

	aliceUser := register(t, alice, "alice@itest.local")
	bobUser := register(t, bob, "bob@itest.local")

	// Session and authentication.
	status, payload := alice.do(http.MethodGet, "/api/auth/me", nil)
	if status != http.StatusOK {
		t.Fatalf("me: status %d, body %s", status, payload)
	}
	if me := decode[userBody](t, payload); me.User.Email != "alice@itest.local" {
		t.Errorf("me email = %q", me.User.Email)
	}
	if status, _ := anonymous.do(http.MethodGet, "/api/auth/me", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous me status = %d, want 401", status)
	}
	if status, _ := anonymous.do(http.MethodGet, "/api/tasks", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous tasks status = %d, want 401", status)
	}

	// Create and list.
	milk := createTask(t, alice, "  Buy milk  ", "  2%  ")
	if milk.Title != "Buy milk" || milk.Description != "2%" {
		t.Errorf("created task = %+v, want trimmed fields", milk)
	}
	ship := createTask(t, alice, "Ship task 6", "")
	bobTask := createTask(t, bob, "Bob only", "")

	aliceTasks := listTasks(t, alice, "")
	if len(aliceTasks) != 2 {
		t.Fatalf("alice sees %d tasks, want 2", len(aliceTasks))
	}
	if aliceTasks[0].ID != ship.ID || aliceTasks[1].ID != milk.ID {
		t.Errorf("alice task order = %v, want newest first", []int64{aliceTasks[0].ID, aliceTasks[1].ID})
	}
	if bobTasks := listTasks(t, bob, ""); len(bobTasks) != 1 || bobTasks[0].ID != bobTask.ID {
		t.Errorf("bob tasks = %+v, want only his own", bobTasks)
	}

	// Toggle and filter.
	status, payload = alice.do(http.MethodPatch, taskPath(milk.ID), map[string]bool{"completed": true})
	if status != http.StatusOK {
		t.Fatalf("patch: status %d, body %s", status, payload)
	}
	updated := decode[taskBody](t, payload).Task
	if !updated.Completed {
		t.Error("completed was not set")
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Errorf("updatedAt %v did not advance past createdAt %v", updated.UpdatedAt, updated.CreatedAt)
	}
	if active := listTasks(t, alice, "?status=active"); len(active) != 1 || active[0].ID != ship.ID {
		t.Errorf("active = %+v, want only %d", active, ship.ID)
	}
	if completed := listTasks(t, alice, "?status=completed"); len(completed) != 1 || completed[0].ID != milk.ID {
		t.Errorf("completed = %+v, want only %d", completed, milk.ID)
	}

	// Validation.
	status, payload = alice.do(http.MethodPatch, taskPath(ship.ID), map[string]string{"title": "   "})
	if status != http.StatusBadRequest {
		t.Errorf("blank title status = %d, want 400", status)
	} else if fields := decode[errorBody](t, payload).Error.Fields; fields["title"] == "" {
		t.Errorf("blank title fields = %v", fields)
	}
	if status, _ := alice.do(http.MethodPatch, taskPath(ship.ID), map[string]any{}); status != http.StatusBadRequest {
		t.Errorf("empty patch status = %d, want 400", status)
	}
	if status, _ := alice.do(http.MethodGet, "/api/tasks?status=archived", nil); status != http.StatusBadRequest {
		t.Errorf("bad status filter = %d, want 400", status)
	}

	// Cross-user access must look like a missing task and change nothing.
	if status, _ := bob.do(http.MethodPatch, taskPath(ship.ID), map[string]bool{"completed": true}); status != http.StatusNotFound {
		t.Errorf("bob patch alice's task = %d, want 404", status)
	}
	if status, _ := bob.do(http.MethodDelete, taskPath(ship.ID), nil); status != http.StatusNotFound {
		t.Errorf("bob delete alice's task = %d, want 404", status)
	}
	after := listTasks(t, alice, "")
	if len(after) != 2 || after[0].Completed {
		t.Errorf("alice tasks after bob's attempts = %+v, want 2 with ship still active", after)
	}

	// Delete, then confirm it is gone.
	if status, _ := alice.do(http.MethodDelete, taskPath(milk.ID), nil); status != http.StatusNoContent {
		t.Errorf("delete status = %d, want 204", status)
	}
	if remaining := listTasks(t, alice, ""); len(remaining) != 1 || remaining[0].ID != ship.ID {
		t.Errorf("remaining = %+v, want only %d", remaining, ship.ID)
	}
	if status, _ := alice.do(http.MethodDelete, taskPath(milk.ID), nil); status != http.StatusNotFound {
		t.Errorf("second delete status = %d, want 404", status)
	}

	// Logout clears the session cookie.
	if status, _ := alice.do(http.MethodPost, "/api/auth/logout", nil); status != http.StatusNoContent {
		t.Errorf("logout status = %d, want 204", status)
	}
	if status, _ := alice.do(http.MethodGet, "/api/auth/me", nil); status != http.StatusUnauthorized {
		t.Errorf("me after logout = %d, want 401", status)
	}

	// The database agrees with the API: one row per user, none crossing over.
	ctx := context.Background()
	var aliceCount, bobCount, crossCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tasks WHERE user_id = $1`, aliceUser.User.ID).Scan(&aliceCount); err != nil {
		t.Fatalf("count alice tasks: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tasks WHERE user_id = $1`, bobUser.User.ID).Scan(&bobCount); err != nil {
		t.Fatalf("count bob tasks: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tasks WHERE user_id = $1 AND id = $2`, aliceUser.User.ID, bobTask.ID).Scan(&crossCount); err != nil {
		t.Fatalf("count cross-user rows: %v", err)
	}
	if aliceCount != 1 || bobCount != 1 {
		t.Errorf("task counts alice=%d bob=%d, want 1 each", aliceCount, bobCount)
	}
	if crossCount != 0 {
		t.Errorf("alice owns %d of bob's tasks, want 0", crossCount)
	}
}

func TestDuplicateEmailHitsUniqueIndex(t *testing.T) {
	server := newTestServer(t)
	first := newClient(t, server)
	second := newClient(t, server)

	register(t, first, "dup@itest.local")

	status, payload := second.do(http.MethodPost, "/api/auth/register",
		map[string]string{"email": "DUP@itest.local", "password": "password123"})
	if status != http.StatusConflict {
		t.Fatalf("duplicate register status = %d, want 409", status)
	}
	if code := decode[errorBody](t, payload).Error.Code; code != "email_taken" {
		t.Errorf("code = %q, want email_taken", code)
	}
}
