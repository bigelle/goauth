package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGetAuthContext(t *testing.T) {
	s := miniredis.RunT(t)
	t.Cleanup(s.Close)

	client := redis.NewClient(&redis.Options{
		Addr:     s.Addr(),
		Protocol: 2,
	})

	cache := RedisCache{
		client: client,
	}

	authCtx := AuthContext{
		UserID:        "someUUID",
		CodeChallenge: "somecodechallenge",
	}
	code := "someauthcodeAABBCC"

	err := cache.StoreAuthContext(context.Background(), code, &authCtx)
	require.NoError(t, err)

	cachedAuthCtx, err := cache.GetDelAuthContext(context.Background(), code)
	require.NoError(t, err)
	require.NotNil(t, cachedAuthCtx)

	// check if some of the values were corrupted or lost:
	require.Equal(t, authCtx, *cachedAuthCtx)

	// check if TTL works:
	s.FastForward(60 * time.Second)
	_, err = cache.GetDelAuthContext(context.Background(), code)
	require.Error(t, err)
}
