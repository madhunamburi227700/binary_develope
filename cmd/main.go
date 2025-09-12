package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/OpsMx/go-app-base/version"
	"github.com/opsmx/ai-gyardian-api/pkg/config"
	"github.com/opsmx/ai-gyardian-api/pkg/handlers"
	"github.com/rs/zerolog/log"
)

// parseFlags will create and parse the CLI flags
// and return the path to be used elsewhere
func parseFlags() (string, error) {
	// string that contains the configured configuration path
	var configPath string

	// set up a CLI flag called "-config" to allow users
	// to supply the configuration file
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")

	// actually parse the flags
	flag.Parse()

	// return the configuration path
	return configPath, nil
}

func main() {
	cfgPath, err := parseFlags()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse config file path")
	}

	err = config.ParseConfig(cfgPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse config")
	}

	log.Info().Msg(version.VersionString())
	if config.GetShowVersion() {
		os.Exit(0)
	}

	log.Info().Msgf("starting appName %s version %s gitBranch %s gitHash %s buildType %s",
		config.GetAppName(), version.VersionString(), version.GitBranch(), version.GitHash(), version.BuildType())

	// Get session Store, default to in-mem
	err = config.AuthenticatorSessionStore()
	if err != nil {
		log.Fatal().Err(err).Msg("error configuring Session store")
	}

	// Setup routes
	router := handlers.SetupRoutes()

	// Setup middleware
	middleware := handlers.SetupMiddleware()
	handler := handlers.ApplyMiddleware(router, middleware...)

	// Start server
	addr := fmt.Sprintf("%s:%s", config.GetApiHost(), config.GetApiPort())
	log.Info().Msgf("Server starting on %s", addr)

	
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
