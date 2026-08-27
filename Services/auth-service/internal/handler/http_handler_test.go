package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/cab-booking/auth-service/internal/domain"
	jwtmgr "github.com/cab-booking/auth-service/internal/jwt"
	"github.com/cab-booking/auth-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func newTestServer() http.Handler {
	repo := repository.NewUserRepository(nil)
	mgr := jwtmgr.NewTokenManager("unit-test-secret-key-32bytes-min!", 15, 7)
	h := NewHTTPHandler(repo, mgr)
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", h.Register)
	mux.HandleFunc("/auth/login", h.Login)
	mux.HandleFunc("/auth/refresh", h.Refresh)
	mux.HandleFunc("/health", h.Health)
	return mux
}

func postJSON(t *testing.T, srv http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status %d", rec.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "ok" || payload["service"] != "auth-service" {
		t.Errorf("unexpected health payload: %v", payload)
	}
}

func TestRegisterLoginRefreshE2E(t *testing.T) {
	srv := newTestServer()
	reg := postJSON(t, srv, "/auth/register", RegisterRequest{
		Email:    "e2e.rider@example.com",
		Phone:    "+15551110001",
		Password: "SecurePass1",
		FullName: "E2E Rider",
		Role:     "RIDER",
	})
	if reg.Code != http.StatusCreated {
		t.Fatalf("register status %d body %s", reg.Code, reg.Body.String())
	}
	var created AuthResponse
	if err := json.Unmarshal(reg.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.AccessToken == "" || created.RefreshToken == "" {
		t.Fatal("missing tokens")
	}
	if created.User == nil || created.User.Role != domain.RoleRider {
		t.Fatalf("unexpected user: %+v", created.User)
	}
	if created.User.PasswordHash != "" {
		t.Fatal("password hash must never be serialized")
	}

	login := postJSON(t, srv, "/auth/login", LoginRequest{
		Email:    "E2E.Rider@example.com",
		Password: "SecurePass1",
	})
	if login.Code != http.StatusOK {
		t.Fatalf("login status %d body %s", login.Code, login.Body.String())
	}

	refresh := postJSON(t, srv, "/auth/refresh", RefreshRequest{RefreshToken: created.RefreshToken})
	if refresh.Code != http.StatusOK {
		t.Fatalf("refresh status %d body %s", refresh.Code, refresh.Body.String())
	}
}

func TestRegisterValidation(t *testing.T) {
	srv := newTestServer()
	cases := []struct {
		name string
		body RegisterRequest
		code int
	}{
		{"missing fields", RegisterRequest{Email: "a@b.com"}, http.StatusBadRequest},
		{"invalid email", RegisterRequest{Email: "not-an-email", Phone: "+1", Password: "SecurePass1", FullName: "A"}, http.StatusBadRequest},
		{"short password", RegisterRequest{Email: "a@b.com", Phone: "+1", Password: "Ab1", FullName: "A"}, http.StatusBadRequest},
		{"letters only password", RegisterRequest{Email: "a@b.com", Phone: "+1", Password: "Password", FullName: "A"}, http.StatusBadRequest},
		{"invalid role", RegisterRequest{Email: "a@b.com", Phone: "+1", Password: "SecurePass1", FullName: "A", Role: "HACKER"}, http.StatusBadRequest},
		{"admin escalation", RegisterRequest{Email: "admin@b.com", Phone: "+155500099", Password: "SecurePass1", FullName: "A", Role: "ADMIN"}, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := postJSON(t, srv, "/auth/register", tc.body)
			if rec.Code != tc.code {
				t.Fatalf("got %d want %d body %s", rec.Code, tc.code, rec.Body.String())
			}
		})
	}
}

func TestDuplicateRegisterConflict(t *testing.T) {
	srv := newTestServer()
	body := RegisterRequest{
		Email:    "dup@example.com",
		Phone:    "+15551112222",
		Password: "SecurePass1",
		FullName: "Dup",
		Role:     "RIDER",
	}
	if code := postJSON(t, srv, "/auth/register", body).Code; code != http.StatusCreated {
		t.Fatalf("first register %d", code)
	}
	if code := postJSON(t, srv, "/auth/register", body).Code; code != http.StatusConflict {
		t.Fatalf("second register %d", code)
	}
}

func TestLoginFailures(t *testing.T) {
	srv := newTestServer()
	postJSON(t, srv, "/auth/register", RegisterRequest{
		Email:    "login@example.com",
		Phone:    "+15551113333",
		Password: "SecurePass1",
		FullName: "Login",
	})
	if rec := postJSON(t, srv, "/auth/login", LoginRequest{Email: "login@example.com", Password: "wrongpass1"}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status %d", rec.Code)
	}
	if rec := postJSON(t, srv, "/auth/login", LoginRequest{Email: "missing@example.com", Password: "SecurePass1"}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unknown user status %d", rec.Code)
	}
}

func TestInactiveUserCannotLogin(t *testing.T) {
	repo := repository.NewUserRepository(nil)
	hash, _ := bcrypt.GenerateFromPassword([]byte("SecurePass1"), bcrypt.MinCost)
	_ = repo.CreateUser(httptest.NewRequest(http.MethodPost, "/", nil).Context(), &domain.User{
		ID:           "disabled-1",
		Email:        "disabled@example.com",
		Phone:        "+15551114444",
		PasswordHash: string(hash),
		FullName:     "Disabled",
		Role:         domain.RoleRider,
		IsActive:     false,
	})
	mgr := jwtmgr.NewTokenManager("unit-test-secret-key-32bytes-min!", 15, 7)
	h := NewHTTPHandler(repo, mgr)
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", h.Login)

	rec := postJSON(t, mux, "/auth/login", LoginRequest{Email: "disabled@example.com", Password: "SecurePass1"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("disabled login status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestRefreshRejectsAccessToken(t *testing.T) {
	srv := newTestServer()
	reg := postJSON(t, srv, "/auth/register", RegisterRequest{
		Email:    "refresh@example.com",
		Phone:    "+15551115555",
		Password: "SecurePass1",
		FullName: "Refresh",
	})
	var created AuthResponse
	_ = json.Unmarshal(reg.Body.Bytes(), &created)
	rec := postJSON(t, srv, "/auth/refresh", RefreshRequest{RefreshToken: created.AccessToken})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("access-as-refresh status %d", rec.Code)
	}
}

func TestSQLInjectionPayloadIsLiteral(t *testing.T) {
	srv := newTestServer()
	payload := RegisterRequest{
		Email:    "inject@example.com",
		Phone:    "'; DROP TABLE users; --",
		Password: "SecurePass1",
		FullName: "Robert'); DROP TABLE students;--",
		Role:     "RIDER",
	}
	rec := postJSON(t, srv, "/auth/register", payload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("injection payload should be stored as a literal, got %d %s", rec.Code, rec.Body.String())
	}
	var created AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.User.Phone != payload.Phone {
		t.Errorf("phone stored as %q", created.User.Phone)
	}
}

func TestXSSNameIsNotInterpreted(t *testing.T) {
	srv := newTestServer()
	name := `<script>alert("xss")</script>`
	rec := postJSON(t, srv, "/auth/register", RegisterRequest{
		Email:    "xss@example.com",
		Phone:    "+15551116666",
		Password: "SecurePass1",
		FullName: name,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `\u003cscript\u003e`) && !strings.Contains(rec.Body.String(), name) {
		t.Fatalf("expected escaped or raw name in JSON, got %s", rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatal("response must stay JSON, not HTML")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestConcurrentRegisterLogin(t *testing.T) {
	srv := newTestServer()
	const n = 40
	var wg sync.WaitGroup
	wg.Add(n)
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			email := fmt.Sprintf("load%d@example.com", i)
			reg := postJSON(t, srv, "/auth/register", RegisterRequest{
				Email:    email,
				Phone:    fmt.Sprintf("+15552%04d", i),
				Password: "SecurePass1",
				FullName: "Load",
			})
			if reg.Code != http.StatusCreated {
				errCh <- fmt.Errorf("register %s: %d", email, reg.Code)
				return
			}
			login := postJSON(t, srv, "/auth/login", LoginRequest{Email: email, Password: "SecurePass1"})
			if login.Code != http.StatusOK {
				errCh <- fmt.Errorf("login %s: %d", email, login.Code)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}
