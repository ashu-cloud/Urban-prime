package handler

import (
	"context"

	jwtmgr "github.com/cab-booking/auth-service/internal/jwt"
	authv1 "github.com/cab-booking/proto/gen/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	authv1.UnimplementedAuthServiceServer
	jwtManager *jwtmgr.TokenManager
}

func NewGRPCHandler(jwtManager *jwtmgr.TokenManager) *GRPCHandler {
	return &GRPCHandler{
		jwtManager: jwtManager,
	}
}

func (h *GRPCHandler) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	claims, err := h.jwtManager.ValidateToken(req.Token)
	if err != nil {
		return &authv1.ValidateTokenResponse{
			Valid: false,
		}, nil
	}

	return &authv1.ValidateTokenResponse{
		Valid:  true,
		UserId: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}
