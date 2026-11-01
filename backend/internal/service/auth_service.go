package service

import (
	"context"
	stderrors "errors"
	"net/mail"
	"strings"

	apperrors "github.com/tinkhmt/shopback-todo/backend/internal/errors"
	"github.com/tinkhmt/shopback-todo/backend/internal/model"
	"github.com/tinkhmt/shopback-todo/backend/internal/repository"
	"github.com/tinkhmt/shopback-todo/backend/internal/util"
)

// AuthService holds the auth business rules. It returns AppErrors, so handlers
// never choose status codes.
type AuthService struct {
	repo   repository.UserRepository
	tokens *util.TokenManager
}

func NewAuthService(repo repository.UserRepository, tokens *util.TokenManager) *AuthService {
	return &AuthService{repo: repo, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*model.User, string, error) {
	email, err := normalizeCredentials(email, password)
	if err != nil {
		return nil, "", err
	}

	hash, err := util.HashPassword(password)
	if err != nil {
		if stderrors.Is(err, util.ErrPasswordTooLong) {
			return nil, "", passwordTooLong()
		}
		return nil, "", apperrors.Internal(err)
	}

	user, err := s.repo.Create(ctx, email, hash)
	if err != nil {
		if stderrors.Is(err, repository.ErrEmailTaken) {
			return nil, "", apperrors.Conflict("email_taken", "an account with this email already exists")
		}
		return nil, "", apperrors.Internal(err)
	}

	token, err := s.tokens.Issue(user.ID)
	if err != nil {
		return nil, "", apperrors.Internal(err)
	}
	return user, token, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	email, err := normalizeCredentials(email, password)
	if err != nil {
		return nil, "", err
	}

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if stderrors.Is(err, repository.ErrUserNotFound) {
			// Same bcrypt cost as a real check: no timing oracle.
			util.CheckPassword(util.DummyPasswordHash, password)
			return nil, "", invalidCredentials()
		}
		return nil, "", apperrors.Internal(err)
	}

	if !util.CheckPassword(user.PasswordHash, password) {
		return nil, "", invalidCredentials()
	}

	token, err := s.tokens.Issue(user.ID)
	if err != nil {
		return nil, "", apperrors.Internal(err)
	}
	return user, token, nil
}

func (s *AuthService) UserByID(ctx context.Context, id int64) (*model.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, repository.ErrUserNotFound) {
			return nil, apperrors.Unauthorized("session is no longer valid")
		}
		return nil, apperrors.Internal(err)
	}
	return user, nil
}

// normalizeCredentials trims and lowercases the email, then validates both
// fields. Passwords are never trimmed.
func normalizeCredentials(email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	fields := map[string]string{}

	if len(email) < 3 || len(email) > 254 {
		fields["email"] = "must be a valid email address"
	} else if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		fields["email"] = "must be a valid email address"
	}

	switch {
	case len(password) < util.MinPasswordLen:
		fields["password"] = "must be at least 8 characters"
	case len(password) > util.MaxPasswordLen:
		fields["password"] = "must be at most 72 bytes"
	}

	if len(fields) > 0 {
		return "", apperrors.Validation(fields)
	}
	return email, nil
}

func invalidCredentials() *apperrors.AppError {
	return apperrors.Unauthorized("invalid email or password")
}

func passwordTooLong() *apperrors.AppError {
	return apperrors.Validation(map[string]string{"password": "must be at most 72 bytes"})
}
