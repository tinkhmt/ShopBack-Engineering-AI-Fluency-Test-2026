package service

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	apperrors "github.com/tinkhmt/shopback-todo/backend/internal/errors"
	"github.com/tinkhmt/shopback-todo/backend/internal/model"
	"github.com/tinkhmt/shopback-todo/backend/internal/repository"
	"github.com/tinkhmt/shopback-todo/backend/internal/util"
)

type fakeRepository struct {
	byEmail map[string]*model.User
	nextID  int64
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byEmail: make(map[string]*model.User)}
}

func (f *fakeRepository) Create(_ context.Context, email, passwordHash string) (*model.User, error) {
	if _, exists := f.byEmail[email]; exists {
		return nil, repository.ErrEmailTaken
	}
	f.nextID++
	user := &model.User{ID: f.nextID, Email: email, PasswordHash: passwordHash, CreatedAt: time.Now()}
	f.byEmail[email] = user
	return user, nil
}

func (f *fakeRepository) FindByEmail(_ context.Context, email string) (*model.User, error) {
	user, ok := f.byEmail[email]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return user, nil
}

func (f *fakeRepository) FindByID(_ context.Context, id int64) (*model.User, error) {
	for _, user := range f.byEmail {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func newTestService() (*AuthService, *fakeRepository) {
	repo := newFakeRepository()
	return NewAuthService(repo, util.NewTokenManager(strings.Repeat("s", 32), time.Minute)), repo
}

func requireAppError(t *testing.T, err error, status int, code string) *apperrors.AppError {
	t.Helper()
	var appErr *apperrors.AppError
	if !stderrors.As(err, &appErr) {
		t.Fatalf("error %v is not *AppError", err)
	}
	if appErr.Status != status {
		t.Errorf("status = %d, want %d", appErr.Status, status)
	}
	if code != "" && appErr.Code != code {
		t.Errorf("code = %q, want %q", appErr.Code, code)
	}
	return appErr
}

func TestRegisterNormalizesEmailAndIssuesToken(t *testing.T) {
	svc, repo := newTestService()

	user, token, err := svc.Register(context.Background(), "  Alice@Example.COM  ", "password123")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", user.Email)
	}
	if repo.byEmail["alice@example.com"] == nil {
		t.Error("user was not stored under the normalized email")
	}
	if !strings.HasPrefix(repo.byEmail["alice@example.com"].PasswordHash, "$2") {
		t.Error("stored password is not a bcrypt hash")
	}

	userID, err := svc.tokens.Parse(token)
	if err != nil {
		t.Fatalf("issued token does not parse: %v", err)
	}
	if userID != user.ID {
		t.Errorf("token subject = %d, want %d", userID, user.ID)
	}
}

func TestRegisterRejectsDuplicateEmailCaseInsensitive(t *testing.T) {
	svc, _ := newTestService()

	if _, _, err := svc.Register(context.Background(), "alice@example.com", "password123"); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	_, _, err := svc.Register(context.Background(), "ALICE@example.com", "password123")
	requireAppError(t, err, 409, "email_taken")
}

func TestRegisterValidatesInput(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		field    string
	}{
		{name: "invalid email", email: "not-an-email", password: "password123", field: "email"},
		{name: "email with display name", email: "Alice <alice@example.com>", password: "password123", field: "email"},
		{name: "short password", email: "alice@example.com", password: "short", field: "password"},
		{name: "password over 72 bytes", email: "alice@example.com", password: strings.Repeat("a", 73), field: "password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService()

			_, _, err := svc.Register(context.Background(), tt.email, tt.password)
			appErr := requireAppError(t, err, 400, "validation_failed")
			if _, ok := appErr.Fields[tt.field]; !ok {
				t.Errorf("fields = %v, want a message for %q", appErr.Fields, tt.field)
			}
		})
	}
}

func TestLoginSucceedsWithNormalizedEmail(t *testing.T) {
	svc, _ := newTestService()

	if _, _, err := svc.Register(context.Background(), "alice@example.com", "password123"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	user, token, err := svc.Login(context.Background(), "ALICE@example.com ", "password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email = %q", user.Email)
	}
	if _, err := svc.tokens.Parse(token); err != nil {
		t.Errorf("issued token does not parse: %v", err)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	svc, _ := newTestService()

	if _, _, err := svc.Register(context.Background(), "alice@example.com", "password123"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, _, err := svc.Login(context.Background(), "alice@example.com", "wrong-password")
	appErr := requireAppError(t, err, 401, "unauthorized")
	if appErr.Message != "invalid email or password" {
		t.Errorf("message = %q", appErr.Message)
	}
}

func TestLoginRejectsUnknownEmail(t *testing.T) {
	svc, _ := newTestService()

	_, _, err := svc.Login(context.Background(), "nobody@example.com", "password123")
	appErr := requireAppError(t, err, 401, "unauthorized")
	if appErr.Message != "invalid email or password" {
		t.Errorf("message = %q", appErr.Message)
	}
}

func TestUserByIDRejectsUnknownUser(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.UserByID(context.Background(), 999)
	requireAppError(t, err, 401, "unauthorized")
}
