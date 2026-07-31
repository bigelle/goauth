package cache

import "context"

type Cache interface {
	StoreAuthCode(ctx context.Context, authCode string, state *UserState) error
}

type UserState struct {
	UserID        string `json:"user_id"`
	CodeChallenge string `json:"code_challenge"`
}
