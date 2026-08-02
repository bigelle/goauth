package cache

import (
	"context"
	"errors"

	"github.com/vmihailenco/msgpack/v5"
)

func NewCache(driver string) (Cache, error) {
	switch driver {
	case "redis":
		return newRedisCache()
	}

	// FIXME:
	return nil, errors.ErrUnsupported
}

type Cache interface {
	StoreAuthContext(ctx context.Context, code string, state *AuthContext) error
	GetDelAuthContext(ctx context.Context, code string) (*AuthContext, error)
	Close() error
}

type AuthContext struct {
	UserID        string `json:"user_id" redis:"user_id"`
	CodeChallenge string `json:"code_challenge" redis:"code_challenge"`
}

type authContextAlias AuthContext

func (c *AuthContext) MarshalBinary() ([]byte, error) {
	return msgpack.Marshal(authContextAlias(*c))
}

func (c *AuthContext) UnmarshalBinary(data []byte) error {
	return msgpack.Unmarshal(data, (*authContextAlias)(c))
}
