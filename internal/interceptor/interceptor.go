package interceptor

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

func UnaryContextServerInterceptor(d time.Duration) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		c, cancel := context.WithTimeout(ctx, d)
		defer cancel()

		return handler(c, req)
	}
}
