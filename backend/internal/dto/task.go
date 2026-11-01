package dto

import "time"

type CreateTaskRequest struct {
	Title       string `json:"title" binding:"required,max=255"`
	Description string `json:"description" binding:"max=2000"`
}

// UpdateTaskRequest uses pointers so an absent field is distinguishable from an
// explicit zero value: {"completed":false} must not look like "not provided".
type UpdateTaskRequest struct {
	Title       *string `json:"title" binding:"omitempty,max=255"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Completed   *bool   `json:"completed"`
}

type TaskResponse struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TaskEnvelope struct {
	Task TaskResponse `json:"task"`
}

type TasksResponse struct {
	Tasks []TaskResponse `json:"tasks"`
}
