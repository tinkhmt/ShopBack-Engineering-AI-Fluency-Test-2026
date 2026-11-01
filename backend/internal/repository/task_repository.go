package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tinkhmt/shopback-todo/backend/internal/model"
)

var ErrTaskNotFound = errors.New("task not found")

// TaskRepository is the persistence boundary for tasks. Every method takes the
// owner ID: ownership is enforced in SQL, never by a prior existence check, so
// a foreign task is indistinguishable from a missing one.
type TaskRepository interface {
	Create(ctx context.Context, userID int64, title, description string) (*model.Task, error)
	List(ctx context.Context, userID int64, completed *bool) ([]model.Task, error)
	Update(ctx context.Context, userID, id int64, title, description *string, completed *bool) (*model.Task, error)
	Delete(ctx context.Context, userID, id int64) error
}

type postgresTaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) TaskRepository {
	return &postgresTaskRepository{pool: pool}
}

const taskColumns = `id, user_id, title, description, completed, created_at, updated_at`

func (r *postgresTaskRepository) Create(ctx context.Context, userID int64, title, description string) (*model.Task, error) {
	task := &model.Task{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO tasks (user_id, title, description)
		 VALUES ($1, $2, $3)
		 RETURNING `+taskColumns,
		userID, title, description,
	).Scan(&task.ID, &task.UserID, &task.Title, &task.Description, &task.Completed, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return task, nil
}

func (r *postgresTaskRepository) List(ctx context.Context, userID int64, completed *bool) ([]model.Task, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+taskColumns+`
		 FROM tasks
		 WHERE user_id = $1 AND ($2::boolean IS NULL OR completed = $2)
		 ORDER BY created_at DESC, id DESC`,
		userID, completed,
	)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	// Never nil: the handler serialises this as [] rather than null.
	tasks := make([]model.Task, 0)
	for rows.Next() {
		var task model.Task
		if err := rows.Scan(&task.ID, &task.UserID, &task.Title, &task.Description,
			&task.Completed, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

func (r *postgresTaskRepository) Update(ctx context.Context, userID, id int64, title, description *string, completed *bool) (*model.Task, error) {
	task := &model.Task{}
	err := r.pool.QueryRow(ctx,
		`UPDATE tasks
		 SET title = COALESCE($3, title),
		     description = COALESCE($4, description),
		     completed = COALESCE($5, completed),
		     updated_at = now()
		 WHERE id = $1 AND user_id = $2
		 RETURNING `+taskColumns,
		id, userID, title, description, completed,
	).Scan(&task.ID, &task.UserID, &task.Title, &task.Description, &task.Completed, &task.CreatedAt, &task.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return task, nil
}

func (r *postgresTaskRepository) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTaskNotFound
	}
	return nil
}
