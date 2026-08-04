package service

import (
	"context"
	"time"

	accountv1 "github.com/bigelle/auth/gen/account/v1"
	"github.com/bigelle/auth/internal/cache"
	"github.com/bigelle/auth/internal/crypt"
	"github.com/bigelle/auth/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewAccountService(db db.Database, cache cache.Cache) *AccountService {
	return &AccountService{
		dbcon: db,
		c:     cache,
	}
}

type AccountService struct {
	accountv1.UnimplementedAccountServiceServer
	c     cache.Cache
	dbcon db.Database
}

func (s *AccountService) CreateAccount(c context.Context, req *accountv1.CreateAccountRequest) (*accountv1.CreateAccountResponse, error) {
	ctx, cancel := context.WithTimeout(c, 30*time.Second)
	defer cancel()

	pass, err := crypt.HashPassword(req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to hash password")
	}

	if err = s.dbcon.CreateUser(ctx, req.Username, req.Email, pass); err != nil {
		// FIXME: handle the error properly
		return nil, status.Error(codes.InvalidArgument, "error storing user in db, idk fix me or some shi")
	}

	return &accountv1.CreateAccountResponse{}, nil
}
