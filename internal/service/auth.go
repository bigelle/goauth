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

func (s *AuthService) AuthenticateAccount(ctx context.Context, req *authv1.AuthenticateAccountRequest) (*authv1.AuthenticateAccountResponse, error) {
	authCode := crypt.MakeAuthCode()

	u, err := s.QueryUser(ctx, req)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "unable to make request to database")
	}

	if !crypt.VerifyPassword(req.Password, u.PasswordHash) {
		return nil, status.Error(codes.Unauthenticated, "wrong username, email or password")
	}

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

func (s *AuthService) ExchangeAuthCode(ctx context.Context, req *authv1.ExchangeAuthCodeRequest) (*authv1.ExchangeAuthCodeResponse, error) {
	now := time.Now()
	expIn := s.RefreshTokenExpiresIn()
	expAt := now.Add(expIn)

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

	// FIXME: read key from config or env
	access, err := makeJwt(authCtx.UserID, now, expAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "can not sign JWT token")
	}

	refresh := uuid.Must(uuid.NewV7()).String()
	refreshID := uuid.Must(uuid.NewV7()).String()

	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "can not begin transaction in refresh_tokens table")
	}

	old, err := s.QueryRefreshTokenByOwnerTX(ctx, tx, QueryRefreshTokenParams{
		OwnerID:       authCtx.UserID,
		StillActiveAt: &now,
	})
	if err != nil {
		// NotFoundError may be expected, if for example this is the first ever exchange request
		// so the user never had any refresh tokens
		if !ent.IsNotFound(err) {
			// FIXME:
			tx.Rollback()
			return nil, status.Error(codes.Internal, "idk fix me later")
		}
	}

	// In case the user already has an active refresh token, it should be revoked first
	if old != nil {
		err = s.RevokeRefreshTokenTX(ctx, tx, RevokeRefreshTokenParams{
			OldTokenHash: old.TokenHash,
			RevokedAt:    now,
		})
		if err != nil {
			tx.Rollback()
			return nil, status.Error(codes.Internal, "can not revoke old jwt token")
		}
	}

	err = s.StoreRefreshTokenTX(ctx, tx, StoreRefreshTokenParams{
		ID: refreshID,
		// FIXME: hash it
		TokenHash: refresh,
		OwnerID:   authCtx.UserID,
		// FIXME:
		FamilyID:  "no use",
		CreatedAt: now,
		ExpiresAt: expAt,
	})
	if err != nil {
		tx.Rollback()
		return nil, status.Error(codes.Internal, "can not store new jwt token")
	}

	tx.Commit()

	return &authv1.ExchangeAuthCodeResponse{
		AccessToken:      access,
		RefreshToken:     refresh,
		ExpiresInSeconds: int64(expIn),
	}, nil
}

func (s *AuthService) RefreshAccessToken(ctx context.Context, req *authv1.RefreshAccessTokenRequest) (*authv1.RefreshAccessTokenResponse, error) {
	now := time.Now()
	expIn := s.RefreshTokenExpiresIn()
	expAt := now.Add(expIn)

	tx, err := s.db.Tx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "can not begin transaction in refresh_tokens table")
	}

	old, err := s.QueryRefreshTokenByTokenHashTX(ctx, tx, QueryRefreshTokenParams{
		// FIXME: hash it
		TokenHash: req.RefreshToken,
		WithOwner: true,
	})
	if err != nil {
		tx.Rollback()
		return nil, status.Error(codes.Unauthenticated, "refresh token not found")
	}
	ownerID := old.Edges.Owner.ID

	if old.RevokedAt != nil {
		// TODO:
		// 1. revoke whole token family
		// 2. no matter what, return unauthorized
		return nil, status.Error(codes.Unauthenticated, "this token is revoked")
	}

	if old.ExpiresAt.Before(now) {
		return nil, status.Error(codes.Unauthenticated, "this token is revoked")
	}

	access, err := makeJwt(ownerID, now, expAt)
	if err != nil {
		return nil, status.Error(codes.Internal, "can not sign JWT token")
	}

	err = s.RevokeRefreshTokenTX(ctx, tx, RevokeRefreshTokenParams{
		OldTokenHash: old.TokenHash,
		RevokedAt:    now,
	})
	if err != nil {
		tx.Rollback()
		return nil, status.Error(codes.Internal, "unable to revoke old refresh token")
	}

	refresh := uuid.Must(uuid.NewV7()).String()
	refreshID := uuid.Must(uuid.NewV7()).String()

	err = s.StoreRefreshTokenTX(ctx, tx, StoreRefreshTokenParams{
		ID: refreshID,
		// FIXME: hash it
		TokenHash: refresh,
		OwnerID:   ownerID,
		FamilyID:  old.FamilyID,
		CreatedAt: now,
		ExpiresAt: expAt,
	})
	if err != nil {
		tx.Rollback()
		return nil, status.Error(codes.Internal, "can not post new refresh token")
	}

	tx.Commit()

	return &authv1.RefreshAccessTokenResponse{
		AccessToken:      access,
		RefreshToken:     refresh,
		ExpiresInSeconds: int64(expIn),
	}, nil
}

func (s *AuthService) QueryUser(c context.Context, req *authv1.AuthenticateAccountRequest) (*ent.User, error) {
	switch req.Credential.(type) {
	case *authv1.AuthenticateAccountRequest_Email:
		return s.db.User.Query().Where(user.Email(req.GetEmail())).Only(c)

	case *authv1.AuthenticateAccountRequest_Username:
		return s.db.User.Query().Where(user.Name(req.GetUsername())).Only(c)
	}
	return nil, status.Error(codes.InvalidArgument, "login credentials are required")
}

type StoreRefreshTokenParams struct {
	ID        string
	TokenHash string
	FamilyID  string
	OwnerID   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (s *AuthService) StoreRefreshTokenTX(c context.Context, tx *ent.Tx, params StoreRefreshTokenParams) error {
	return tx.RefreshToken.Create().
		SetID(params.ID).
		SetTokenHash(params.TokenHash).
		SetFamilyID(params.FamilyID).
		SetOwnerID(params.OwnerID).
		SetCreatedAt(params.CreatedAt).
		SetExpiresAt(params.ExpiresAt).
		Exec(c)
}

type RevokeRefreshTokenParams struct {
	OldTokenHash string
	RevokedAt    time.Time
}

func (s *AuthService) RevokeRefreshTokenTX(c context.Context, tx *ent.Tx, params RevokeRefreshTokenParams) error {
	return tx.RefreshToken.Update().
		Where(refreshtoken.TokenHash(params.OldTokenHash)).
		SetRevokedAt(params.RevokedAt).
		Exec(c)
}

type QueryRefreshTokenParams struct {
	OwnerID       string
	TokenHash     string
	StillActiveAt *time.Time
	WithOwner     bool
}

func (s *AuthService) QueryRefreshTokenByOwnerTX(c context.Context, tx *ent.Tx, params QueryRefreshTokenParams) (*ent.RefreshToken, error) {
	q := tx.RefreshToken.Query().
		Where(refreshtoken.HasOwnerWith(user.ID(params.OwnerID)))

	return s.queryRefreshTokenTX(c, q, params)
}

func (s *AuthService) QueryRefreshTokenByTokenHashTX(c context.Context, tx *ent.Tx, params QueryRefreshTokenParams) (*ent.RefreshToken, error) {
	q := tx.RefreshToken.Query().
		Where(refreshtoken.TokenHash(params.TokenHash))

	return s.queryRefreshTokenTX(c, q, params)
}

func (s *AuthService) queryRefreshTokenTX(c context.Context, q *ent.RefreshTokenQuery, params QueryRefreshTokenParams) (*ent.RefreshToken, error) {
	if params.StillActiveAt != nil {
		q.Where(refreshtoken.Not(refreshtoken.RevokedAtGTE(*params.StillActiveAt)))
	}

	if params.WithOwner {
		q.WithOwner()
	}

	return q.Only(c)
}

func (s *AuthService) RefreshTokenExpiresIn() time.Duration {
	// FIXME: read from config
	return 30 * 24 * time.Hour
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
