package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tinkhmt/shopback-todo/backend/internal/model"
)

var (
	ErrEmailTaken   = errors.New("email already registered")
	ErrUserNotFound = errors.New("user not found")
)

// UserRepository is the persistence boundary for users.
type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id int64) (*model.User, error)
}

type postgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &postgresUserRepository{pool: pool}
}

const uniqueViolationCode = "23505"

func (r *postgresUserRepository) Create(ctx context.Context, email, passwordHash string) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, email, password_hash, created_at`,
		email, passwordHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		// The unique index is the source of truth, so concurrent registrations
		// cannot both succeed.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (r *postgresUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.findOne(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE lower(email) = $1`,
		email)
}

func (r *postgresUserRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
	return r.findOne(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = $1`,
		id)
}

func (r *postgresUserRepository) findOne(ctx context.Context, query string, arg any) (*model.User, error) {
	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}
