package main

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/bigelle/auth/ent"
	accountv1 "github.com/bigelle/auth/gen/account/v1"
	authv1 "github.com/bigelle/auth/gen/auth/v1"
	"github.com/bigelle/auth/internal/cache"
	"github.com/bigelle/auth/internal/interceptor"
	"github.com/bigelle/auth/internal/service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// TODO: read a config from yaml(?)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "file:/tmp/app.db?cache=shared&_fk=1" // текущий дефолт остаётся как есть для локального go run
	}

	// FIXME: don't use hardcoded options
	log.Info().Msg("setting up database")
	log.Debug().Str("driver", "sqlite3").Str("options", dsn).Msg("database options")

	db, err := ent.Open("sqlite3", dsn)
	db = db.Debug()
	if err != nil {
		log.Fatal().AnErr("database error", err).Msg("error opening database connection")
	}
	defer db.Close()

	if err := db.Schema.Create(context.Background()); err != nil {
		log.Fatal().AnErr("database error", err).Msg("failed migrating schema")
	}

	// FIXME: don't use hardcoded options
	log.Info().Msg("setting up cache")
	log.Debug().Str("driver", "redis").Msg("cache options")
	c, err := cache.NewCache("redis")
	if err != nil {
		log.Fatal().AnErr("cache error", err).Msg("error opening cache connection")
	}
	defer c.Close()

	// FIXME: don't use hardcoded options
	log.Info().Int("port", 50051).Msg("opening socket on port")
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal().AnErr("socket error", err).Msg("error opening tcp socket")
	}
	log.Info().Int("port", 50052).Msg("listening on port")

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// FIXME: read timeout from config
			interceptor.UnaryContextServerInterceptor(30*time.Second),
			interceptor.UnaryLoggingServerInterceptor(),
		),
	)

	authService := service.NewAuthService(db, c)
	authv1.RegisterAuthServiceServer(server, authService)

	accountService := service.NewAccountService(db, c)
	accountv1.RegisterAccountServiceServer(server, accountService)

	reflection.Register(server)

	log.Info().Msg("server is online")
	if err := server.Serve(listener); err != nil {
		log.Fatal().AnErr("grpc server error", err).Msg("error serving grpc server on tcp socket")
	}
}
