package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cab-booking/auth-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository handles SQL queries with an in-memory sync fallback
type UserRepository struct {
	pool         *pgxpool.Pool
	mu           sync.RWMutex
	usersByEmail map[string]*domain.User
	usersByID    map[string]*domain.User
	usersByPhone map[string]*domain.User
	tokens       map[string]*domain.RefreshToken
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	repo := &UserRepository{
		pool:         pool,
		usersByEmail: make(map[string]*domain.User),
		usersByID:    make(map[string]*domain.User),
		usersByPhone: make(map[string]*domain.User),
		tokens:       make(map[string]*domain.RefreshToken),
	}

	// Seed default demo accounts
	passHash, _ := bcrypt.GenerateFromPassword([]byte("SecurePassword123!"), bcrypt.DefaultCost)
	now := time.Now()

	rider := &domain.User{
		ID:           "rid_001",
		Email:        "alexander.vance@urbanprime.com",
		Phone:        "+15553456789",
		PasswordHash: string(passHash),
		FullName:     "Alexander Vance",
		Role:         domain.RoleRider,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	driver := &domain.User{
		ID:           "drv_901",
		Email:        "marcus.sterling@driver.urbanprime.com",
		Phone:        "+15559876543",
		PasswordHash: string(passHash),
		FullName:     "Marcus Sterling",
		Role:         domain.RoleDriver,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	repo.saveInMemory(rider)
	repo.saveInMemory(driver)

	return repo
}

func (r *UserRepository) SetPool(pool *pgxpool.Pool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pool = pool
}

func (r *UserRepository) saveInMemory(u *domain.User) {
	r.mu.Lock()
	defer r.mu.Unlock()
	emailKey := strings.ToLower(strings.TrimSpace(u.Email))
	r.usersByEmail[emailKey] = u
	r.usersByID[u.ID] = u
	if u.Phone != "" {
		r.usersByPhone[u.Phone] = u
	}
}

func (r *UserRepository) CreateUser(ctx context.Context, u *domain.User) error {
	emailKey := strings.ToLower(strings.TrimSpace(u.Email))

	r.mu.RLock()
	if _, exists := r.usersByEmail[emailKey]; exists {
		r.mu.RUnlock()
		return fmt.Errorf("user with email %s already exists", u.Email)
	}
	if u.Phone != "" {
		if _, exists := r.usersByPhone[u.Phone]; exists {
			r.mu.RUnlock()
			return fmt.Errorf("user with phone %s already exists", u.Phone)
		}
	}
	r.mu.RUnlock()

	// Try inserting into PostgreSQL if pool is available
	r.mu.RLock()
	pool := r.pool
	r.mu.RUnlock()

	if pool != nil {
		query := `
			INSERT INTO users (user_id, email, phone, password_hash, full_name, role, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
		_, err := pool.Exec(ctx, query, u.ID, emailKey, u.Phone, u.PasswordHash, u.FullName, string(u.Role), u.IsActive, u.CreatedAt, u.UpdatedAt)
		if err != nil {
			// If DB fails due to duplicate key, return error
			if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
				return fmt.Errorf("user already exists: %w", err)
			}
		}
	}

	// Always sync in-memory
	r.saveInMemory(u)
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	emailKey := strings.ToLower(strings.TrimSpace(email))

	r.mu.RLock()
	pool := r.pool
	r.mu.RUnlock()

	if pool != nil {
		query := `
			SELECT user_id, email, phone, password_hash, full_name, role, is_active, created_at, updated_at
			FROM users
			WHERE LOWER(email) = $1
		`
		var u domain.User
		var roleStr string
		err := pool.QueryRow(ctx, query, emailKey).Scan(
			&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.FullName, &roleStr, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
		)
		if err == nil {
			u.Role = domain.UserRole(roleStr)
			r.saveInMemory(&u)
			return &u, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			// Non-fatal error, check in-memory
		}
	}

	// Fallback to in-memory store
	r.mu.RLock()
	defer r.mu.RUnlock()
	if u, ok := r.usersByEmail[emailKey]; ok {
		return u, nil
	}

	return nil, fmt.Errorf("user not found with email: %s", email)
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	pool := r.pool
	r.mu.RUnlock()

	if pool != nil {
		query := `
			SELECT user_id, email, phone, password_hash, full_name, role, is_active, created_at, updated_at
			FROM users
			WHERE user_id = $1
		`
		var u domain.User
		var roleStr string
		err := pool.QueryRow(ctx, query, id).Scan(
			&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.FullName, &roleStr, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
		)
		if err == nil {
			u.Role = domain.UserRole(roleStr)
			r.saveInMemory(&u)
			return &u, nil
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if u, ok := r.usersByID[id]; ok {
		return u, nil
	}

	return nil, fmt.Errorf("user not found with id: %s", id)
}

func (r *UserRepository) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	r.mu.Lock()
	r.tokens[token.TokenHash] = token
	pool := r.pool
	r.mu.Unlock()

	if pool != nil {
		query := `
			INSERT INTO refresh_tokens (token_id, user_id, token_hash, expires_at, revoked, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (token_hash) DO NOTHING
		`
		_, _ = pool.Exec(ctx, query, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.Revoked, token.CreatedAt)
	}
	return nil
}

func (r *UserRepository) RevokeUserRefreshTokens(ctx context.Context, userID string) error {
	r.mu.Lock()
	for _, t := range r.tokens {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	pool := r.pool
	r.mu.Unlock()

	if pool != nil {
		query := `UPDATE refresh_tokens SET revoked = true WHERE user_id = $1`
		_, _ = pool.Exec(ctx, query, userID)
	}
	return nil
}

