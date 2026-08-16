package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// UserClaims defines custom JWT payload claims
type UserClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// TokenManager signs and verifies JWT access & refresh tokens
type TokenManager struct {
	secretKey     []byte
	accessTTLMin  time.Duration
	refreshTTLDays time.Duration
}

func NewTokenManager(secretKey string, accessTTLMin, refreshTTLDays int) *TokenManager {
	return &TokenManager{
		secretKey:      []byte(secretKey),
		accessTTLMin:   time.Duration(accessTTLMin) * time.Minute,
		refreshTTLDays: time.Duration(refreshTTLDays) * 24 * time.Hour,
	}
}

// GeneratePair creates a signed JWT Access Token and Refresh Token pair
func (m *TokenManager) GeneratePair(userID, email, role string) (accessToken string, refreshToken string, err error) {
	now := time.Now()

	// 1. Create Access Token (short-lived)
	accessClaims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTLMin)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: userID,
		Email:  email,
		Role:   role,
	}
	accToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accToken.SignedString(m.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	// 2. Create Refresh Token (long-lived)
	refreshClaims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTLDays)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: userID,
		Email:  email,
		Role:   role,
	}
	refToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refToken.SignedString(m.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// ValidateToken parses and validates a signed JWT string
func (m *TokenManager) ValidateToken(tokenStr string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
