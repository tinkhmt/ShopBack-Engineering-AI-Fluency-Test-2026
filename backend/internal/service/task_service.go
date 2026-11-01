package service

import (
	"context"
	stderrors "errors"
	"strings"
	"unicode/utf8"

	apperrors "github.com/tinkhmt/shopback-todo/backend/internal/errors"
	"github.com/tinkhmt/shopback-todo/backend/internal/model"
	"github.com/tinkhmt/shopback-todo/backend/internal/repository"
)

const (
	maxTitleLen       = 255
	maxDescriptionLen = 2000
)

// TaskService holds the task business rules: validation, filter parsing, and
// translation of repository sentinels into HTTP-safe errors.
type TaskService struct {
	repo repository.TaskRepository
}

func NewTaskService(repo repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) Create(ctx context.Context, userID int64, title, description string) (*model.Task, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)

	fields := map[string]string{}
	if n := utf8.RuneCountInString(title); n < 1 || n > maxTitleLen {
		fields["title"] = "must be between 1 and 255 characters"
	}
	if utf8.RuneCountInString(description) > maxDescriptionLen {
		fields["description"] = "must be at most 2000 characters"
	}
	if len(fields) > 0 {
		return nil, apperrors.Validation(fields)
	}

	task, err := s.repo.Create(ctx, userID, title, description)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return task, nil
}

// List maps the status query parameter to a filter. "" and "all" mean no filter.
func (s *TaskService) List(ctx context.Context, userID int64, status string) ([]model.Task, error) {
	completed, err := parseStatus(status)
	if err != nil {
		return nil, err
	}

	tasks, err := s.repo.List(ctx, userID, completed)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return tasks, nil
}

// Update applies only the provided fields; nil means "leave unchanged".
func (s *TaskService) Update(ctx context.Context, userID, id int64, title, description *string, completed *bool) (*model.Task, error) {
	if title == nil && description == nil && completed == nil {
		return nil, apperrors.Validation(map[string]string{
			"body": "at least one of title, description or completed is required",
		})
	}

	fields := map[string]string{}
	if title != nil {
		trimmed := strings.TrimSpace(*title)
		if n := utf8.RuneCountInString(trimmed); n < 1 || n > maxTitleLen {
			fields["title"] = "must be between 1 and 255 characters"
		}
		title = &trimmed
	}
	if description != nil {
		trimmed := strings.TrimSpace(*description)
		if utf8.RuneCountInString(trimmed) > maxDescriptionLen {
			fields["description"] = "must be at most 2000 characters"
		}
		description = &trimmed
	}
	if len(fields) > 0 {
		return nil, apperrors.Validation(fields)
	}

	task, err := s.repo.Update(ctx, userID, id, title, description, completed)
	if err != nil {
		if stderrors.Is(err, repository.ErrTaskNotFound) {
			return nil, taskNotFound()
		}
		return nil, apperrors.Internal(err)
	}
	return task, nil
}

func (s *TaskService) Delete(ctx context.Context, userID, id int64) error {
	err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		if stderrors.Is(err, repository.ErrTaskNotFound) {
			return taskNotFound()
		}
		return apperrors.Internal(err)
	}
	return nil
}

func parseStatus(status string) (*bool, error) {
	switch status {
	case "", "all":
		return nil, nil
	case "active":
		active := false
		return &active, nil
	case "completed":
		done := true
		return &done, nil
	default:
		return nil, apperrors.Validation(map[string]string{
			"status": "must be one of all, active, completed",
		})
	}
}

func taskNotFound() *apperrors.AppError {
	return apperrors.NotFound("task not found")
}
