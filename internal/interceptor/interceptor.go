package interceptor

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
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

func UnaryLoggingServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		start := time.Now()

		resp, err = handler(ctx, req)
		if err != nil {
			s, ok := status.FromError(err)
			if ok {
				log.Error().
					Str("code", s.Code().String()).
					Str("message", s.Message()).
					Str("method", info.FullMethod).
					Dur("duration", time.Since(start)).
					Send()
			}
		} else {
			log.Info().
				Str("code", "OK").
				Str("method", info.FullMethod).
				Dur("duration", time.Since(start)).
				Send()
		}

		return resp, err
	}
}

func UnaryPanicServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Any("cause", r).Str("method", info.FullMethod).Msg("PANIC")
			}
		}()

		resp, err = handler(ctx, req)

		return resp, err
	}
}
