package domain

import (
	"fmt"
	"time"
)

// UserRole defines custom string enum for user roles
type UserRole string

const (
	RoleRider  UserRole = "RIDER"
	RoleDriver UserRole = "DRIVER"
	RoleAdmin  UserRole = "ADMIN"
)

// User represents the core entity for registered platform users
type User struct {
	ID           string    `json:"user_id"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	PasswordHash string    `json:"-"` // Hidden from JSON serialization!
	FullName     string    `json:"full_name"`
	Role         UserRole  `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RefreshToken represents a stored refresh token entry in PostgreSQL
type RefreshToken struct {
	ID        string    `json:"token_id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

func (r UserRole) IsValid() bool {
	switch r {
	case RoleRider, RoleDriver, RoleAdmin:
		return true
	default:
		return false
	}
}

func (u *User) Validate() error {
	if u.Email == "" || u.Phone == "" || u.FullName == "" {
		return fmt.Errorf("email, phone, and full_name are required")
	}
	if !u.Role.IsValid() {
		return fmt.Errorf("invalid user role: %s", u.Role)
	}
	return nil
}
