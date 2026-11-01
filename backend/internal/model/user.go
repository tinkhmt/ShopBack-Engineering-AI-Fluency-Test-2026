package model

import "time"

// User is the domain model. It carries no transport tags; DTOs live in dto.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
