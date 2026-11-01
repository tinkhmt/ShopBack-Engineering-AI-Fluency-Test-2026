package dto

import "time"

// CredentialsRequest is the body for both register and login.
// Email format is validated in the service after trimming; binding only
// enforces shape so "  user@example.com " is accepted and normalized.
type CredentialsRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type UserResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type AuthResponse struct {
	User UserResponse `json:"user"`
}
