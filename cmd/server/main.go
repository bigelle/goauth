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
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// TODO: read a config from yaml(?)

	// FIXME: don't use hardcoded options
	db, err := db.NewDatabase(db.SQLITE3_driver, ":memory:")
	if err != nil {
		log.Fatal().AnErr("database error", err).Msg("error opening database connection")
	}

	cache := cache.NewCache("redis")

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal().AnErr("socket error", err).Msg("error opening tcp socket")
	}

	service := service.NewAuthService(db, cache)

	server := grpc.NewServer()
	authv1.RegisterAuthServiceServer(server, service)

	if err := server.Serve(listener); err != nil {
		log.Fatal().AnErr("grpc server error", err).Msg("error serving grpc server on tcp socket")
	}
}
