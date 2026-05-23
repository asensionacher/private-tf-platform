package models

import "time"

// User represents a web UI user
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // "admin" or "reader"
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserCreate is used by admins to create a new user
type UserCreate struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

// UserUpdate is used by admins to update a user
type UserUpdate struct {
	Username *string `json:"username,omitempty"`
	Role     *string `json:"role,omitempty"`
}

// UserChangePassword is used to change a user's password
type UserChangePassword struct {
	Password string `json:"password" binding:"required"`
}

// LoginRequest holds credentials for login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse returns the JWT and user info
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
