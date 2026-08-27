package handler

import (
	"context"
	"testing"

	jwtmgr "github.com/cab-booking/auth-service/internal/jwt"
	authv1 "github.com/cab-booking/proto/gen/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCValidateToken(t *testing.T) {
	mgr := jwtmgr.NewTokenManager("grpc-test-secret-key-32bytes-min!", 15, 7)
	h := NewGRPCHandler(mgr)
	access, _, err := mgr.GeneratePair("user-9", "g@example.com", "DRIVER")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := h.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{Token: access})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Valid || resp.UserId != "user-9" || resp.Role != "DRIVER" {
		t.Errorf("unexpected response %+v", resp)
	}

	invalid, err := h.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{Token: "nope"})
	if err != nil {
		t.Fatal(err)
	}
	if invalid.Valid {
		t.Fatal("garbage token must be invalid")
	}

	_, err = h.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{Token: ""})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty token code %v", status.Code(err))
	}
}
