package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/bigelle/auth/ent"
	accountv1 "github.com/bigelle/auth/gen/account/v1"
	authv1 "github.com/bigelle/auth/gen/auth/v1"
	"github.com/bigelle/auth/internal/cache"
	"github.com/bigelle/auth/internal/config"
	"github.com/bigelle/auth/internal/interceptor"
	"github.com/bigelle/auth/internal/service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	_ "github.com/mattn/go-sqlite3"
)

func ConnectDatabase(cfg *config.DatabaseConfig) (*ent.Client, error) {
	dsn, err := makeDsn(cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid database options")
	}

	return ent.Open(cfg.Driver, dsn)
}

func makeDsn(cfg *config.DatabaseConfig) (dsn string, err error) {
	switch cfg.Driver {
	case "sqlite3":
		dsn = makeSqliteDsn(&cfg.Sqlite)
	}

	if dsn == "" {
		err = fmt.Errorf("unsupported database: %s", cfg.Driver)
	}
	fmt.Println("dsn:", dsn)

	return dsn, err
}

func makeSqliteDsn(cfg *config.SqliteConfig) string {
	builder := strings.Builder{}
	builder.WriteString("file:")
	builder.WriteString(cfg.File)
	builder.WriteRune('?')

	builder.WriteString("cache=")
	builder.WriteString(cfg.Cache)

	builder.WriteString("&_fk=1")

	return builder.String()
}

func Migrate(db *ent.Client) error {
	return db.Schema.Create(context.Background())
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	cfg, err := config.LoadConfig(os.Getenv("CONFIG_FILE_PATH"))
	if err != nil {
		log.Fatal().Err(err).Msg("error loading config")
	}

	db, err := ConnectDatabase(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("error connecting to database")
	}

	if err = Migrate(db); err != nil {
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
	log.Info().Int("port", cfg.Server.Port).Msg("opening socket on port")
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal().AnErr("socket error", err).Msg("error opening tcp socket")
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.UnaryPanicServerInterceptor(),
			interceptor.UnaryLoggingServerInterceptor(),
			// FIXME: read timeout from config
			interceptor.UnaryContextServerInterceptor(30*time.Second),
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
