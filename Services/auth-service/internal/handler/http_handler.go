package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cab-booking/auth-service/internal/domain"
	jwtmgr "github.com/cab-booking/auth-service/internal/jwt"
	"github.com/cab-booking/auth-service/internal/repository"
	"github.com/cab-booking/pkg/logger"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type HTTPHandler struct {
	repo       *repository.UserRepository
	jwtManager *jwtmgr.TokenManager
}

func NewHTTPHandler(repo *repository.UserRepository, jwtManager *jwtmgr.TokenManager) *HTTPHandler {
	return &HTTPHandler{
		repo:       repo,
		jwtManager: jwtManager,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Role     string `json:"role"` // "RIDER", "DRIVER", "ADMIN"
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         *domain.User `json:"user"`
}

func (h *HTTPHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.Phone == "" || req.FullName == "" {
		http.Error(w, "email, password, phone, and full_name are required", http.StatusBadRequest)
		return
	}

	role := domain.UserRole(req.Role)
	if req.Role == "" {
		role = domain.RoleRider
	}

	// Hash password using bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		Role:         role,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := user.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.CreateUser(r.Context(), user); err != nil {
		logger.Error(r.Context(), "User registration failed", "error", err)
		http.Error(w, "Email or phone already exists", http.StatusConflict)
		return
	}

	// Issue JWT pair
	accToken, refToken, err := h.jwtManager.GeneratePair(user.ID, user.Email, string(user.Role))
	if err != nil {
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, AuthResponse{
		AccessToken:  accToken,
		RefreshToken: refToken,
		User:         user,
	})
}

func (h *HTTPHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.repo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	accToken, refToken, err := h.jwtManager.GeneratePair(user.ID, user.Email, string(user.Role))
	if err != nil {
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		AccessToken:  accToken,
		RefreshToken: refToken,
		User:         user,
	})
}

func (h *HTTPHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	claims, err := h.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	user, err := h.repo.GetByID(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	accToken, refToken, err := h.jwtManager.GeneratePair(user.ID, user.Email, string(user.Role))
	if err != nil {
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		AccessToken:  accToken,
		RefreshToken: refToken,
		User:         user,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
