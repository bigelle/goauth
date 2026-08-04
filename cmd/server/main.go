package main

import (
	"net"

	authv1 "github.com/bigelle/auth/gen/auth/v1"
	"github.com/bigelle/auth/internal/cache"
	"github.com/bigelle/auth/internal/db"
	"github.com/bigelle/auth/internal/service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// TODO: read a config from yaml(?)

	// FIXME: don't use hardcoded options
	log.Info().Msg("setting up database")
	log.Debug().Str("driver", db.SQLITE3_driver).Str("options", ":memory").Msg("database options")
	db, err := db.NewDatabase(db.SQLITE3_driver, ":memory:")
	if err != nil {
		log.Fatal().AnErr("database error", err).Msg("error opening database connection")
	}

	// FIXME: don't use hardcoded options
	log.Info().Msg("setting up cache")
	log.Debug().Str("driver", "redis").Msg("cache options")
	c, err := cache.NewCache("redis")
	if err != nil {
		log.Fatal().AnErr("cache error", err).Msg("error opening cache connection")
	}

	// FIXME: don't use hardcoded options
	log.Info().Int("port", 50051).Msg("opening socket on port")
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal().AnErr("socket error", err).Msg("error opening tcp socket")
	}
	log.Info().Int("port", 50051).Msg("listening on port")

	service := service.NewAuthService(db, c)

	server := grpc.NewServer()
	authv1.RegisterAuthServiceServer(server, service)
	reflection.Register(server)

	log.Info().Msg("server is online")
	if err := server.Serve(listener); err != nil {
		log.Fatal().AnErr("grpc server error", err).Msg("error serving grpc server on tcp socket")
	}
}
