package cache

import (
	"context"
	"time"

	"github.com/bigelle/auth/internal/crypt"
	"github.com/redis/go-redis/v9"
)

func newRedisCache() (*RedisCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis-server:6379",
		Password: "",
		DB:       0,
		Protocol: 2,
	})

	return &RedisCache{client: rdb}, nil
}

type RedisCache struct {
	client *redis.Client
}

func (c *RedisCache) StoreAuthContext(ctx context.Context, code string, authCtx *AuthContext) error {
	key := authCodeKey(code)
	return c.client.Set(ctx, key, authCtx, 60*time.Second).Err()
}

func (c *RedisCache) GetDelAuthContext(ctx context.Context, code string) (*AuthContext, error) {
	key := authCodeKey(code)

	var authCtx AuthContext
	if err := c.client.GetDel(ctx, key).Scan(&authCtx); err != nil {
		return nil, err
	}

	return &authCtx, nil
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

func authCodeKey(code string) string {
	return "authcode:" + crypt.HashAuthCodeSHA256(code)
}
