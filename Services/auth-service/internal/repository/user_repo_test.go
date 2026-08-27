package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cab-booking/auth-service/internal/domain"
)

func newUser(id, email, phone string) *domain.User {
	now := time.Now()
	return &domain.User{
		ID:           id,
		Email:        email,
		Phone:        phone,
		PasswordHash: "hash",
		FullName:     "Test User",
		Role:         domain.RoleRider,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestCreateUserAndGetByEmailCaseInsensitive(t *testing.T) {
	repo := NewUserRepository(nil)
	ctx := context.Background()

	u := newUser("u1", "Rider.One@Example.COM", "+15550001")
	if err := repo.CreateUser(ctx, u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	got, err := repo.GetByEmail(ctx, "rider.one@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.ID != "u1" {
		t.Errorf("got id %s", got.ID)
	}

	if _, err := repo.GetByID(ctx, "u1"); err != nil {
		t.Fatalf("GetByID: %v", err)
	}
}

func TestCreateUserDuplicateEmailAndPhone(t *testing.T) {
	repo := NewUserRepository(nil)
	ctx := context.Background()

	if err := repo.CreateUser(ctx, newUser("u1", "dup@example.com", "+15550002")); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateUser(ctx, newUser("u2", "dup@example.com", "+15550003")); err == nil {
		t.Fatal("duplicate email must fail")
	}
	if err := repo.CreateUser(ctx, newUser("u3", "other@example.com", "+15550002")); err == nil {
		t.Fatal("duplicate phone must fail")
	}
}

func TestGetByEmailUnknown(t *testing.T) {
	repo := NewUserRepository(nil)
	if _, err := repo.GetByEmail(context.Background(), "missing@example.com"); err == nil {
		t.Fatal("expected not found")
	}
}

func TestSeededDemoAccountsExist(t *testing.T) {
	repo := NewUserRepository(nil)
	if _, err := repo.GetByEmail(context.Background(), "alexander.vance@urbanprime.com"); err != nil {
		t.Fatalf("seeded rider missing: %v", err)
	}
	if _, err := repo.GetByEmail(context.Background(), "marcus.sterling@driver.urbanprime.com"); err != nil {
		t.Fatalf("seeded driver missing: %v", err)
	}
}

func TestConcurrentUniqueRegistration(t *testing.T) {
	repo := NewUserRepository(nil)
	ctx := context.Background()
	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			errCh <- repo.CreateUser(ctx, newUser(
				fmt.Sprintf("id-%d", i),
				"same@example.com",
				fmt.Sprintf("+1555%04d", i),
			))
		}(i)
	}
	wg.Wait()
	close(errCh)

	success := 0
	for err := range errCh {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("expected exactly 1 successful registration, got %d", success)
	}
}

func TestConcurrentDistinctRegistrations(t *testing.T) {
	repo := NewUserRepository(nil)
	ctx := context.Background()
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			errCh <- repo.CreateUser(ctx, newUser(
				fmt.Sprintf("id-%d", i),
				fmt.Sprintf("user%d@example.com", i),
				fmt.Sprintf("+1556%04d", i),
			))
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("distinct registration failed: %v", err)
		}
	}
}
