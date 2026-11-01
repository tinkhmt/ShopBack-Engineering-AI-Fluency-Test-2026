package model

import "time"

// Task is the domain model. It carries no transport tags; DTOs live in dto.
type Task struct {
	ID          int64
	UserID      int64
	Title       string
	Description string
	Completed   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
