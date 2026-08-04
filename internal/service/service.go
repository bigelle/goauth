package service

import (
	"context"
	"errors"
	"time"

	authv1 "github.com/bigelle/auth/gen/auth/v1"
	"github.com/bigelle/auth/internal/cache"
	"github.com/bigelle/auth/internal/crypt"
	"github.com/bigelle/auth/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewAuthService(db db.Database, cache cache.Cache) *AuthService {
	return &AuthService{db: db, cache: cache}
}

type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	cache cache.Cache
	db    db.Database
}

func (s *AuthService) Register(c context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	ctx, cancel := context.WithTimeout(c, 30*time.Second)
	defer cancel()

	pass, err := crypt.HashPassword(req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to hash the password")
	}

	if err = s.db.CreateUser(ctx, req.Username, req.Email, pass); err != nil {
		// FIXME: handle errors properly
		return nil, status.Error(codes.InvalidArgument, "bad request")
	}

	return &authv1.RegisterResponse{}, nil
}

func (s *AuthService) Login(c context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	var (
		user *db.User
		err  error
	)

	ctx, cancel := context.WithTimeout(c, 30*time.Second)
	defer cancel()

	switch req.Credential.(type) {
	case *authv1.LoginRequest_Email:
		user, err = s.db.GetUserByEmail(ctx, req.GetEmail())

	case *authv1.LoginRequest_Username:
		user, err = s.db.GetUserByName(ctx, req.GetUsername())

	case nil:
		return nil, status.Error(codes.InvalidArgument, "login credentials are required")
	}

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		// TODO: check for timeouts
		return nil, status.Error(codes.Internal, "unable to make request to database")
	}

	if !crypt.VerifyPassword(req.Password, user.PasswordHash) {
		return nil, status.Error(codes.Unauthenticated, "wrong username, email or password")
	}

	authCode := crypt.MakeAuthCode()

	if err := s.cache.StoreAuthContext(ctx, authCode, &cache.AuthContext{
		UserID:        user.UUID,
		CodeChallenge: req.GetChallenge(),
	}); err != nil {
		// TODO: check for conflicts(?)
		return nil, status.Error(codes.Internal, "unable to store auth code in cache")
	}

	return &authv1.LoginResponse{
		AuthCode: &authCode,
	}, nil
}
