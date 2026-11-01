package handler_test

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tinkhmt/shopback-todo/backend/internal/model"
	"github.com/tinkhmt/shopback-todo/backend/internal/repository"
)

type memoryTaskRepository struct {
	byID   map[int64]*model.Task
	nextID int64
}

func newMemoryTaskRepository() *memoryTaskRepository {
	return &memoryTaskRepository{byID: make(map[int64]*model.Task)}
}

func (m *memoryTaskRepository) Create(_ context.Context, userID int64, title, description string) (*model.Task, error) {
	m.nextID++
	now := time.Now()
	task := &model.Task{
		ID:          m.nextID,
		UserID:      userID,
		Title:       title,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.byID[task.ID] = task
	return task, nil
}

func (m *memoryTaskRepository) List(_ context.Context, userID int64, completed *bool) ([]model.Task, error) {
	tasks := make([]model.Task, 0)
	for _, task := range m.byID {
		if task.UserID != userID {
			continue
		}
		if completed != nil && task.Completed != *completed {
			continue
		}
		tasks = append(tasks, *task)
	}
	// Mirror the SQL order: newest first.
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID > tasks[j].ID })
	return tasks, nil
}

func (m *memoryTaskRepository) Update(_ context.Context, userID, id int64, title, description *string, completed *bool) (*model.Task, error) {
	task, ok := m.byID[id]
	if !ok || task.UserID != userID {
		return nil, repository.ErrTaskNotFound
	}
	if title != nil {
		task.Title = *title
	}
	if description != nil {
		task.Description = *description
	}
	if completed != nil {
		task.Completed = *completed
	}
	task.UpdatedAt = time.Now()
	return task, nil
}

func (m *memoryTaskRepository) Delete(_ context.Context, userID, id int64) error {
	task, ok := m.byID[id]
	if !ok || task.UserID != userID {
		return repository.ErrTaskNotFound
	}
	delete(m.byID, id)
	return nil
}

type taskEnvelope struct {
	Task struct {
		ID          int64     `json:"id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Completed   bool      `json:"completed"`
		CreatedAt   time.Time `json:"createdAt"`
		UpdatedAt   time.Time `json:"updatedAt"`
	} `json:"task"`
}

type tasksEnvelope struct {
	Tasks []struct {
		ID        int64  `json:"id"`
		Title     string `json:"title"`
		Completed bool   `json:"completed"`
	} `json:"tasks"`
}

func createTask(t *testing.T, r *gin.Engine, cookie *http.Cookie, body string) taskEnvelope {
	t.Helper()
	w := perform(r, http.MethodPost, "/api/tasks", body, cookie)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body %s", w.Code, w.Body.String())
	}
	var created taskEnvelope
	decodeJSON(t, w, &created)
	return created
}

func TestTaskRoutesRequireAuthentication(t *testing.T) {
	r := newTestRouter(fakePinger{})

	tests := []struct{ method, path, body string }{
		{http.MethodGet, "/api/tasks", ""},
		{http.MethodPost, "/api/tasks", `{"title":"x"}`},
		{http.MethodPatch, "/api/tasks/1", `{"completed":true}`},
		{http.MethodDelete, "/api/tasks/1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			w := perform(r, tt.method, tt.path, tt.body, nil)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", w.Code)
			}
		})
	}
}

func TestCreateTask(t *testing.T) {
	r := newTestRouter(fakePinger{})
	cookie := registerUser(t, r)

	created := createTask(t, r, cookie, `{"title":"  Buy milk  ","description":"  2%  "}`)

	if created.Task.ID == 0 {
		t.Error("id is zero")
	}
	if created.Task.Title != "Buy milk" {
		t.Errorf("title = %q, want trimmed", created.Task.Title)
	}
	if created.Task.Description != "2%" {
		t.Errorf("description = %q, want trimmed", created.Task.Description)
	}
	if created.Task.Completed {
		t.Error("new task is completed")
	}
}

func TestCreateTaskValidation(t *testing.T) {
	r := newTestRouter(fakePinger{})
	cookie := registerUser(t, r)

	w := perform(r, http.MethodPost, "/api/tasks", `{"title":"   "}`, cookie)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	body := decodeError(t, w)
	if body.Error.Code != "validation_failed" {
		t.Errorf("code = %q", body.Error.Code)
	}
	if body.Error.Fields["title"] == "" {
		t.Errorf("fields = %v, want a title message", body.Error.Fields)
	}
}

func TestListTasksIsEmptyArray(t *testing.T) {
	r := newTestRouter(fakePinger{})
	cookie := registerUser(t, r)

	w := perform(r, http.MethodGet, "/api/tasks", "", cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != `{"tasks":[]}` {
		t.Errorf("body = %s, want {\"tasks\":[]}", got)
	}
}

func TestUpdateTaskAndFilter(t *testing.T) {
	r := newTestRouter(fakePinger{})
	cookie := registerUser(t, r)

	first := createTask(t, r, cookie, `{"title":"first"}`)
	second := createTask(t, r, cookie, `{"title":"second"}`)

	w := perform(r, http.MethodPatch, "/api/tasks/"+itoa(first.Task.ID), `{"completed":true}`, cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("patch status = %d, body %s", w.Code, w.Body.String())
	}
	var updated taskEnvelope
	decodeJSON(t, w, &updated)
	if !updated.Task.Completed {
		t.Error("completed was not set")
	}

	tests := []struct {
		status string
		wantID int64
	}{
		{status: "active", wantID: second.Task.ID},
		{status: "completed", wantID: first.Task.ID},
	}

	for _, tt := range tests {
		t.Run("status="+tt.status, func(t *testing.T) {
			w := perform(r, http.MethodGet, "/api/tasks?status="+tt.status, "", cookie)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
			}
			var body tasksEnvelope
			decodeJSON(t, w, &body)
			if len(body.Tasks) != 1 || body.Tasks[0].ID != tt.wantID {
				t.Fatalf("tasks = %+v, want only id %d", body.Tasks, tt.wantID)
			}
		})
	}

	w = perform(r, http.MethodGet, "/api/tasks?status=archived", "", cookie)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400", w.Code)
	}
}

func TestUpdateTaskKeepsUnprovidedFields(t *testing.T) {
	r := newTestRouter(fakePinger{})
	cookie := registerUser(t, r)

	created := createTask(t, r, cookie, `{"title":"original","description":"keep me"}`)

	w := perform(r, http.MethodPatch, "/api/tasks/"+itoa(created.Task.ID), `{"title":"renamed"}`, cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}

	var updated taskEnvelope
	decodeJSON(t, w, &updated)
	if updated.Task.Title != "renamed" {
		t.Errorf("title = %q", updated.Task.Title)
	}
	if updated.Task.Description != "keep me" {
		t.Errorf("description = %q, want unchanged", updated.Task.Description)
	}
}

func TestEmptyPatchIsRejected(t *testing.T) {
	r := newTestRouter(fakePinger{})
	cookie := registerUser(t, r)

	created := createTask(t, r, cookie, `{"title":"task"}`)

	w := perform(r, http.MethodPatch, "/api/tasks/"+itoa(created.Task.ID), `{}`, cookie)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if code := decodeError(t, w).Error.Code; code != "validation_failed" {
		t.Errorf("code = %q", code)
	}
}

func TestInvalidTaskIDIsRejected(t *testing.T) {
	r := newTestRouter(fakePinger{})
	cookie := registerUser(t, r)

	for _, path := range []string{"/api/tasks/abc", "/api/tasks/0", "/api/tasks/-1"} {
		t.Run(path, func(t *testing.T) {
			w := perform(r, http.MethodDelete, path, "", cookie)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
			if code := decodeError(t, w).Error.Code; code != "invalid_id" {
				t.Errorf("code = %q, want invalid_id", code)
			}
		})
	}
}

func TestForeignTaskIsNotFound(t *testing.T) {
	r := newTestRouter(fakePinger{})
	alice := registerUser(t, r)
	bob := registerEmail(t, r, "bob@example.com")

	created := createTask(t, r, alice, `{"title":"alice only"}`)
	path := "/api/tasks/" + itoa(created.Task.ID)

	w := perform(r, http.MethodGet, "/api/tasks", "", bob)
	if w.Code != http.StatusOK {
		t.Fatalf("bob list status = %d", w.Code)
	}
	var list tasksEnvelope
	decodeJSON(t, w, &list)
	if len(list.Tasks) != 0 {
		t.Fatalf("bob sees %d tasks, want 0", len(list.Tasks))
	}

	w = perform(r, http.MethodPatch, path, `{"completed":true}`, bob)
	if w.Code != http.StatusNotFound {
		t.Fatalf("bob patch status = %d, want 404", w.Code)
	}

	w = perform(r, http.MethodDelete, path, "", bob)
	if w.Code != http.StatusNotFound {
		t.Fatalf("bob delete status = %d, want 404", w.Code)
	}

	// The owner's task must be untouched by the foreign attempts.
	w = perform(r, http.MethodGet, "/api/tasks", "", alice)
	var after tasksEnvelope
	decodeJSON(t, w, &after)
	if len(after.Tasks) != 1 || after.Tasks[0].Completed {
		t.Fatalf("alice tasks = %+v, want one incomplete task", after.Tasks)
	}
}

func TestDeleteTask(t *testing.T) {
	r := newTestRouter(fakePinger{})
	cookie := registerUser(t, r)

	created := createTask(t, r, cookie, `{"title":"delete me"}`)
	path := "/api/tasks/" + itoa(created.Task.ID)

	if w := perform(r, http.MethodDelete, path, "", cookie); w.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", w.Code)
	}
	if w := perform(r, http.MethodDelete, path, "", cookie); w.Code != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want 404", w.Code)
	}
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}
