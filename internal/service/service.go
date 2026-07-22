package service

import (
	"context"

	authnv1 "github.com/bigelle/authn/gen/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService struct {
	authnv1.UnimplementedAuthServiceServer
}

func (s *AuthService) Login(ctx context.Context, req *authnv1.LoginRequest) (*authnv1.LoginResponse, error) {
	switch req.Credential.(type) {

	case *authnv1.LoginRequest_Email:
		return s.loginByEmail(ctx, req)

	case *authnv1.LoginRequest_Username:
		return s.loginByUsername(ctx, req)

	}

	return nil, status.Error(codes.InvalidArgument, "login credentials are required")
}

func (s *AuthService) loginByEmail(ctx context.Context, req *authnv1.LoginRequest) (*authnv1.LoginResponse, error) {
	return nil, nil
}
func (s *AuthService) loginByUsername(ctx context.Context, req *authnv1.LoginRequest) (*authnv1.LoginResponse, error) {
	return nil, nil
}
