package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/cab-booking/auth-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository handles raw SQL queries for user authentication in PostgreSQL
type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) CreateUser(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (user_id, email, phone, password_hash, full_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query, u.ID, u.Email, u.Phone, u.PasswordHash, u.FullName, string(u.Role), u.IsActive, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT user_id, email, phone, password_hash, full_name, role, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var u domain.User
	var roleStr string
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.FullName, &roleStr, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found with email: %s", email)
		}
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}
	u.Role = domain.UserRole(roleStr)
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT user_id, email, phone, password_hash, full_name, role, is_active, created_at, updated_at
		FROM users
		WHERE user_id = $1
	`
	var u domain.User
	var roleStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.FullName, &roleStr, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found with id: %s", id)
		}
		return nil, fmt.Errorf("failed to query user by id: %w", err)
	}
	u.Role = domain.UserRole(roleStr)
	return &u, nil
}

func (r *UserRepository) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (token_id, user_id, token_hash, expires_at, revoked, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.Revoked, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}
	return nil
}

func (r *UserRepository) RevokeUserRefreshTokens(ctx context.Context, userID string) error {
	query := `UPDATE refresh_tokens SET revoked = true WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh tokens: %w", err)
	}
	return nil
}
