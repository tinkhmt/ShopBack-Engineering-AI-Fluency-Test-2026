package service

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tinkhmt/shopback-todo/backend/internal/model"
	"github.com/tinkhmt/shopback-todo/backend/internal/repository"
)

type fakeTaskRepository struct {
	byID   map[int64]*model.Task
	nextID int64
}

func newFakeTaskRepository() *fakeTaskRepository {
	return &fakeTaskRepository{byID: make(map[int64]*model.Task)}
}

func (f *fakeTaskRepository) Create(_ context.Context, userID int64, title, description string) (*model.Task, error) {
	f.nextID++
	now := time.Now()
	task := &model.Task{
		ID:          f.nextID,
		UserID:      userID,
		Title:       title,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	f.byID[task.ID] = task
	return task, nil
}

func (f *fakeTaskRepository) List(_ context.Context, userID int64, completed *bool) ([]model.Task, error) {
	tasks := make([]model.Task, 0)
	for _, task := range f.byID {
		if task.UserID != userID {
			continue
		}
		if completed != nil && task.Completed != *completed {
			continue
		}
		tasks = append(tasks, *task)
	}
	// Deterministic order: the real query sorts by created_at DESC, id DESC.
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks, nil
}

func (f *fakeTaskRepository) Update(_ context.Context, userID, id int64, title, description *string, completed *bool) (*model.Task, error) {
	task, ok := f.byID[id]
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

func (f *fakeTaskRepository) Delete(_ context.Context, userID, id int64) error {
	task, ok := f.byID[id]
	if !ok || task.UserID != userID {
		return repository.ErrTaskNotFound
	}
	delete(f.byID, id)
	return nil
}

func newTaskTestService() (*TaskService, *fakeTaskRepository) {
	repo := newFakeTaskRepository()
	return NewTaskService(repo), repo
}

func TestCreateTaskTrimsFields(t *testing.T) {
	svc, _ := newTaskTestService()

	task, err := svc.Create(context.Background(), 7, "  Buy milk  ", "  2%  ")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if task.Title != "Buy milk" {
		t.Errorf("title = %q, want %q", task.Title, "Buy milk")
	}
	if task.Description != "2%" {
		t.Errorf("description = %q, want %q", task.Description, "2%")
	}
	if task.UserID != 7 {
		t.Errorf("userID = %d, want 7", task.UserID)
	}
	if task.Completed {
		t.Error("new task is completed")
	}
}

func TestCreateTaskValidation(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		field       string
	}{
		{name: "empty title", title: "", field: "title"},
		{name: "whitespace title", title: "   ", field: "title"},
		{name: "title over 255 runes", title: strings.Repeat("a", 256), field: "title"},
		{name: "description over 2000 runes", title: "ok", description: strings.Repeat("a", 2001), field: "description"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTaskTestService()

			_, err := svc.Create(context.Background(), 1, tt.title, tt.description)
			appErr := requireAppError(t, err, 400, "validation_failed")
			if _, ok := appErr.Fields[tt.field]; !ok {
				t.Errorf("fields = %v, want a message for %q", appErr.Fields, tt.field)
			}
		})
	}
}

func TestCreateTaskAccepts255RuneTitle(t *testing.T) {
	svc, _ := newTaskTestService()

	if _, err := svc.Create(context.Background(), 1, strings.Repeat("é", 255), ""); err != nil {
		t.Fatalf("255-rune title rejected: %v", err)
	}
}

func TestListTaskStatusFilter(t *testing.T) {
	svc, repo := newTaskTestService()
	ctx := context.Background()

	active, err := svc.Create(ctx, 1, "active", "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	done, err := svc.Create(ctx, 1, "done", "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	repo.byID[done.ID].Completed = true

	tests := []struct {
		status string
		want   []int64
	}{
		{status: "", want: []int64{active.ID, done.ID}},
		{status: "all", want: []int64{active.ID, done.ID}},
		{status: "active", want: []int64{active.ID}},
		{status: "completed", want: []int64{done.ID}},
	}

	for _, tt := range tests {
		t.Run("status="+tt.status, func(t *testing.T) {
			tasks, err := svc.List(ctx, 1, tt.status)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if len(tasks) != len(tt.want) {
				t.Fatalf("got %d tasks, want %d", len(tasks), len(tt.want))
			}
			for i, id := range tt.want {
				if tasks[i].ID != id {
					t.Errorf("tasks[%d].ID = %d, want %d", i, tasks[i].ID, id)
				}
			}
		})
	}
}

func TestListTaskRejectsUnknownStatus(t *testing.T) {
	svc, _ := newTaskTestService()

	_, err := svc.List(context.Background(), 1, "archived")
	appErr := requireAppError(t, err, 400, "validation_failed")
	if _, ok := appErr.Fields["status"]; !ok {
		t.Errorf("fields = %v, want a message for status", appErr.Fields)
	}
}

func TestListTaskIsScopedToUser(t *testing.T) {
	svc, _ := newTaskTestService()
	ctx := context.Background()

	if _, err := svc.Create(ctx, 1, "mine", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := svc.Create(ctx, 2, "theirs", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	tasks, err := svc.List(ctx, 1, "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "mine" {
		t.Fatalf("tasks = %+v, want only the caller's task", tasks)
	}
}

func TestUpdateTaskAppliesOnlyProvidedFields(t *testing.T) {
	svc, _ := newTaskTestService()
	ctx := context.Background()

	created, err := svc.Create(ctx, 1, "original", "keep me")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	title := "renamed"
	updated, err := svc.Update(ctx, 1, created.ID, &title, nil, nil)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Title != "renamed" {
		t.Errorf("title = %q", updated.Title)
	}
	if updated.Description != "keep me" {
		t.Errorf("description = %q, want unchanged", updated.Description)
	}
	if updated.Completed {
		t.Error("completed changed unexpectedly")
	}
}

func TestUpdateTaskCanClearDescription(t *testing.T) {
	svc, _ := newTaskTestService()
	ctx := context.Background()

	created, err := svc.Create(ctx, 1, "task", "remove me")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	empty := ""
	updated, err := svc.Update(ctx, 1, created.ID, nil, &empty, nil)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Description != "" {
		t.Errorf("description = %q, want empty", updated.Description)
	}
}

func TestUpdateTaskRejectsEmptyPatch(t *testing.T) {
	svc, _ := newTaskTestService()

	_, err := svc.Update(context.Background(), 1, 1, nil, nil, nil)
	appErr := requireAppError(t, err, 400, "validation_failed")
	if _, ok := appErr.Fields["body"]; !ok {
		t.Errorf("fields = %v, want a message for body", appErr.Fields)
	}
}

func TestUpdateTaskValidation(t *testing.T) {
	svc, _ := newTaskTestService()
	ctx := context.Background()

	created, err := svc.Create(ctx, 1, "task", "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	blank := "   "
	_, err = svc.Update(ctx, 1, created.ID, &blank, nil, nil)
	appErr := requireAppError(t, err, 400, "validation_failed")
	if _, ok := appErr.Fields["title"]; !ok {
		t.Errorf("fields = %v, want a message for title", appErr.Fields)
	}
}

func TestUpdateAndDeleteRejectUnknownOrForeignTask(t *testing.T) {
	svc, _ := newTaskTestService()
	ctx := context.Background()

	created, err := svc.Create(ctx, 1, "alice task", "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	done := true
	_, err = svc.Update(ctx, 2, created.ID, nil, nil, &done)
	requireAppError(t, err, 404, "not_found")

	err = svc.Delete(ctx, 2, created.ID)
	requireAppError(t, err, 404, "not_found")

	if err := svc.Delete(ctx, 1, created.ID); err != nil {
		t.Fatalf("Delete() by owner error = %v", err)
	}
	if err := svc.Delete(ctx, 1, created.ID); err == nil {
		t.Fatal("deleting a missing task succeeded")
	}
}
