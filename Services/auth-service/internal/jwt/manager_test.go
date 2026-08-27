package jwt

import (
	"strings"
	"sync"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidatePair(t *testing.T) {
	mgr := NewTokenManager("test-secret-key-at-least-32-bytes!", 15, 7)

	access, refresh, err := mgr.GeneratePair("user-1", "rider@example.com", "RIDER")
	if err != nil {
		t.Fatalf("GeneratePair: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatal("expected non-empty tokens")
	}
	if access == refresh {
		t.Fatal("access and refresh tokens must differ")
	}

	claims, err := mgr.ValidateToken(access)
	if err != nil {
		t.Fatalf("ValidateToken access: %v", err)
	}
	if claims.UserID != "user-1" || claims.Email != "rider@example.com" || claims.Role != "RIDER" {
		t.Errorf("unexpected access claims: %+v", claims)
	}
	if claims.TokenType != TokenTypeAccess {
		t.Errorf("access token_type = %q", claims.TokenType)
	}

	refreshClaims, err := mgr.ValidateRefreshToken(refresh)
	if err != nil {
		t.Fatalf("ValidateRefreshToken: %v", err)
	}
	if refreshClaims.TokenType != TokenTypeRefresh {
		t.Errorf("refresh token_type = %q", refreshClaims.TokenType)
	}
}

func TestAccessTokenCannotBeUsedAsRefresh(t *testing.T) {
	mgr := NewTokenManager("test-secret-key-at-least-32-bytes!", 15, 7)
	access, _, err := mgr.GeneratePair("user-1", "a@b.com", "RIDER")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.ValidateRefreshToken(access); err == nil {
		t.Fatal("access token must not validate as a refresh token")
	}
}

func TestRejectsWrongSecret(t *testing.T) {
	issuer := NewTokenManager("issuer-secret-aaaaaaaaaaaaaaaa", 15, 7)
	verifier := NewTokenManager("other-secret-bbbbbbbbbbbbbbbb", 15, 7)
	access, _, err := issuer.GeneratePair("user-1", "a@b.com", "RIDER")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.ValidateToken(access); err == nil {
		t.Fatal("token signed with a different secret must be rejected")
	}
}

func TestRejectsNoneAlgorithm(t *testing.T) {
	mgr := NewTokenManager("test-secret-key-at-least-32-bytes!", 15, 7)
	claims := UserClaims{
		RegisteredClaims: jwtv5.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Hour)),
		},
		UserID:    "user-1",
		Email:     "attacker@example.com",
		Role:      "ADMIN",
		TokenType: TokenTypeAccess,
	}
	unsigned := jwtv5.NewWithClaims(jwtv5.SigningMethodNone, claims)
	token, err := unsigned.SignedString(jwtv5.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}
	if _, err := mgr.ValidateToken(token); err == nil {
		t.Fatal("alg=none tokens must be rejected")
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	mgr := NewTokenManager("test-secret-key-at-least-32-bytes!", 0, 7)
	// Zero-minute TTL expires at issuance; wait so the clock moves past expiry.
	access, _, err := mgr.GeneratePair("user-1", "a@b.com", "RIDER")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(15 * time.Millisecond)
	if _, err := mgr.ValidateToken(access); err == nil {
		t.Fatal("expired access token must be rejected")
	}
}

func TestTamperedPayloadRejected(t *testing.T) {
	mgr := NewTokenManager("test-secret-key-at-least-32-bytes!", 15, 7)
	access, _, err := mgr.GeneratePair("user-1", "a@b.com", "RIDER")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(access, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}
	parts[1] = parts[1] + "x"
	if _, err := mgr.ValidateToken(strings.Join(parts, ".")); err == nil {
		t.Fatal("tampered JWT must be rejected")
	}
}

func TestEmptyTokenRejected(t *testing.T) {
	mgr := NewTokenManager("test-secret-key-at-least-32-bytes!", 15, 7)
	if _, err := mgr.ValidateToken(""); err == nil {
		t.Fatal("empty token must be rejected")
	}
	if _, err := mgr.ValidateToken("not-a-jwt"); err == nil {
		t.Fatal("garbage token must be rejected")
	}
}

func TestConcurrentTokenRoundTrip(t *testing.T) {
	mgr := NewTokenManager("test-secret-key-at-least-32-bytes!", 15, 7)
	const n = 200
	var wg sync.WaitGroup
	wg.Add(n)
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			access, refresh, err := mgr.GeneratePair("user", "a@b.com", "RIDER")
			if err != nil {
				errCh <- err
				return
			}
			if _, err := mgr.ValidateToken(access); err != nil {
				errCh <- err
				return
			}
			if _, err := mgr.ValidateRefreshToken(refresh); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent JWT failure: %v", err)
	}
}
