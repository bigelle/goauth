package main

import (
	"github.com/bigelle/auth"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	cfg, err := auth.SetupConfig()
	if err != nil {
		log.Err(err).Msg("failed to setup config")
	}
	_ = cfg // FIXME:
}
