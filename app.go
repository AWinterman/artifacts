package main

import (
	"artifacts/src"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

func main() {
	s, err := artifacts.LoadSpecification()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err != nil {
		log.Fatal().Msgf("Failed to load config %v\n", err)
	}

	session, err := artifacts.NewStorage(s)
	if err != nil {
		log.Fatal().Msgf("Failed to connect to db %v\n", err)
	}

	aux := artifacts.NewCodeArtifactAux(s)

	if s.Load {
		log.Info().Fields(s).Msg("Importing artifact lists from AWS codeartifact")
		go artifacts.LoadArtifacts(err, aux, s, session)
	}
	artifacts.LoadTemplates(s)
	server := artifacts.NewServer(s.Listen, session)
	artifacts.StartServer(server)
}
