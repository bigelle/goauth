package main

import (
	"net"

	authnv1 "github.com/bigelle/authn/gen/auth/v1"
	"github.com/bigelle/authn/internal/service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal().AnErr("socket error", err).Msg("error opening tcp socket")
	}

	server := grpc.NewServer()
	authnv1.RegisterAuthServiceServer(server, &service.AuthService{})

	if err := server.Serve(listener); err != nil {
		log.Fatal().AnErr("grpc server error", err).Msg("error serving grpc server on tcp socket")
	}
}
