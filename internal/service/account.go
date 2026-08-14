package service

import (
	"context"
	"time"

	"github.com/bigelle/auth/ent"
	accountv1 "github.com/bigelle/auth/gen/account/v1"
	"github.com/bigelle/auth/internal/cache"
	"github.com/bigelle/auth/internal/crypt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewAccountService(db *ent.Client, cache cache.Cache) *AccountService {
	return &AccountService{
		db: db,
		c:  cache,
	}
}

type AccountService struct {
	accountv1.UnimplementedAccountServiceServer
	c  cache.Cache
	db *ent.Client
}

func (s *AccountService) CreateAccount(c context.Context, req *accountv1.CreateAccountRequest) (*accountv1.CreateAccountResponse, error) {
	ctx, cancel := context.WithTimeout(c, 30*time.Second)
	defer cancel()

	pass, err := crypt.HashPassword(req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, "unable to hash password")
	}

	uuid := uuid.Must(uuid.NewV7())

	_, err = s.db.User.Create().
		SetID(uuid.String()).
		SetName(req.Username).
		SetEmail(req.Email).
		SetPasswordHash(pass).
		Save(ctx)
	if err != nil {
		log.Err(err).Msg("error creating user")
		if ent.IsValidationError(err) {
			return nil, status.Error(codes.InvalidArgument, "bad request")
		}
		if ent.IsConstraintError(err) {
			// NOTE: what if it was a uuid collision?
			// that still could happen, could'nt it?
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
		return nil, status.Error(codes.Internal, "internal error while creating user")
	}

	return &accountv1.CreateAccountResponse{}, nil
}
