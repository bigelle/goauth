package service

import (
	"context"
	"time"

	"github.com/bigelle/auth/ent"
	"github.com/bigelle/auth/ent/refreshtoken"
	"github.com/bigelle/auth/ent/user"
	authv1 "github.com/bigelle/auth/gen/auth/v1"
	"github.com/bigelle/auth/internal/cache"
	"github.com/bigelle/auth/internal/crypt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewAuthService(db *ent.Client, cache cache.Cache) *AuthService {
	return &AuthService{db: db, cache: cache}
}

type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	cache cache.Cache
	db    *ent.Client
}

func (s *AuthService) AuthenticateAccount(c context.Context, req *authv1.AuthenticateAccountRequest) (*authv1.AuthenticateAccountResponse, error) {
	var (
		u   *ent.User
		err error
	)

	ctx, cancel := context.WithTimeout(c, 30*time.Second)
	defer cancel()

	switch req.Credential.(type) {
	case *authv1.AuthenticateAccountRequest_Email:
		u, err = s.db.User.Query().Where(user.Email(req.GetEmail())).Only(ctx)

	case *authv1.AuthenticateAccountRequest_Username:
		u, err = s.db.User.Query().Where(user.Name(req.GetUsername())).Only(ctx)

	case nil:
		return nil, status.Error(codes.InvalidArgument, "login credentials are required")
	}

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "unable to make request to database")
	}

	if !crypt.VerifyPassword(req.Password, u.PasswordHash) {
		return nil, status.Error(codes.Unauthenticated, "wrong username, email or password")
	}

	authCode := crypt.MakeAuthCode()

	if err := s.cache.StoreAuthContext(ctx, authCode, &cache.AuthContext{
		UserID:        u.ID,
		CodeChallenge: req.GetChallenge(),
	}); err != nil {
		// TODO: check for conflicts(?)
		return nil, status.Error(codes.Internal, "unable to store auth code in cache")
	}

	return &authv1.AuthenticateAccountResponse{
		AuthCode: &authCode,
	}, nil
}

func (s *AuthService) ExchangeAuthCode(c context.Context, req *authv1.ExchangeAuthCodeRequest) (*authv1.ExchangeAuthCodeResponse, error) {
	ctx, cancel := context.WithTimeout(c, 30*time.Second)
	defer cancel()

	// NOTE: maybe I should use GET instead of GETDEL
	// because there's 2 (or 3) points where there could be an internal error
	// and the token would be lost.
	// BUT ALSO I should DEL the auth code right before sending response
	// and check if it really was deleted by this function call
	authCtx, err := s.cache.GetDelAuthContext(ctx, req.Code)
	if err != nil {
		// FIXME: handle the error properly
		return nil, status.Error(codes.Internal, "unable to get auth context from cache")
	}

	if !crypt.SolvePKCEChallenge(authCtx.CodeChallenge, req.Verifier) {
		return nil, status.Error(codes.Unauthenticated, "can not solve challenge with given verifier")
	}

	now := time.Now()
	exp := now.Add(30 * 24 * time.Hour) // FIXME: read it from config

	// FIXME: read key from config or env
	access, err := makeJwt(authCtx.UserID, now, exp)
	if err != nil {
		return nil, status.Error(codes.Internal, "can not sign JWT token")
	}

	refresh := uuid.Must(uuid.NewV7())
	refreshID := uuid.Must(uuid.NewV7())

	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "can not begin transaction in refresh_tokens table")
	}

	oldToken, err := tx.RefreshToken.Query().Where(
		refreshtoken.HasOwnerWith(user.ID(authCtx.UserID)),
		refreshtoken.Not(refreshtoken.RevokedAtGTE(now)),
	).Only(ctx)

	if err != nil {
		if !ent.IsNotFound(err) {
			// FIXME:
			tx.Rollback()
			return nil, status.Error(codes.Internal, "idk fix me later")
		}
	}

	if oldToken != nil {
		if err = oldToken.Update().SetRevokedAt(now).Exec(ctx); err != nil {
			tx.Rollback()
			return nil, status.Error(codes.Internal, "can not revoke old jwt token")
		}
	}

	err = tx.RefreshToken.Create().
		SetID(refreshID.String()).
		// FIXME: hash it
		SetTokenHash(refresh.String()).
		SetOwnerID(authCtx.UserID).
		//FIXME: implement family id
		SetFamilyID("no use").
		SetCreatedAt(now).
		SetExpiresAt(exp).
		Exec(ctx)
	if err != nil {
		tx.Rollback()
		return nil, status.Error(codes.Internal, "can not store new jwt token")
	}

	tx.Commit()

	return &authv1.ExchangeAuthCodeResponse{
		AccessToken:      access,
		RefreshToken:     refresh.String(),
		ExpiresInSeconds: int64(30 * 24 * time.Hour),
	}, nil
}

func (s *AuthService) RefreshAccessToken(c context.Context, req *authv1.RefreshAccessTokenRequest) (*authv1.RefreshAccessTokenResponse, error) {
	ctx, cancel := context.WithTimeout(c, 30*time.Second)
	defer cancel()

	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "can not begin transaction in refresh_tokens table")
	}

	old, err := tx.RefreshToken.Query().
		// FIXME: hash it
		Where(refreshtoken.TokenHash(req.RefreshToken)).
		WithOwner().
		Only(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "refresh token not found")
	}

	if old.RevokedAt != nil {
		// TODO:
		// 1. revoke whole token family
		// 2. no matter what, return unauthorized
		return nil, status.Error(codes.Unauthenticated, "this token is revoked")
	}

	now := time.Now()
	if old.ExpiresAt.Before(now) {
		return nil, status.Error(codes.Unauthenticated, "this token is revoked")
	}

	// FIXME: read from config
	exp := now.Add(30 * 24 * time.Hour)

	access, err := makeJwt(old.Edges.Owner.ID, now, exp)
	if err != nil {
		return nil, status.Error(codes.Internal, "can not sign JWT token")
	}

	err = tx.RefreshToken.Update().
		Where(refreshtoken.TokenHash(req.RefreshToken)).
		SetRevokedAt(now).
		Exec(ctx)
	if err != nil {
		tx.Rollback()
		return nil, status.Error(codes.Internal, "can not revoke old refresh token")
	}

	refresh := uuid.Must(uuid.NewV7())
	refreshID := uuid.Must(uuid.NewV7())

	err = tx.RefreshToken.Create().
		SetID(refreshID.String()).
		SetTokenHash(refresh.String()).
		SetFamilyID(old.FamilyID).
		SetOwnerID(old.Edges.Owner.ID).
		SetCreatedAt(now).
		SetExpiresAt(exp).
		Exec(ctx)
	if err != nil {
		tx.Rollback()
		return nil, status.Error(codes.Internal, "can not post new refresh token")
	}

	tx.Commit()

	return &authv1.RefreshAccessTokenResponse{
		AccessToken:      access,
		RefreshToken:     refresh.String(),
		ExpiresInSeconds: int64(30 * 24 * time.Hour),
	}, nil
}

func makeJwt(userID string, now, exp time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": jwt.NewNumericDate(now),
		"nbf": jwt.NewNumericDate(now),
		"exp": jwt.NewNumericDate(exp),
		"jti": uuid.Must(uuid.NewV7()).String(),
		// private claims:
		// FIXME: currently claims are only using user UUID
		// but in the future there would also be something else
		// so don't forget to use it too
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// FIXME: read key from config or env
	tokenStr, err := token.SignedString([]byte("SECRET"))
	if err != nil {
		return "", status.Error(codes.Internal, "can not sign JWT token")
	}

	return tokenStr, nil
}
