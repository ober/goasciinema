package cmd

import (
	"fmt"
	"os"

	"github.com/ober/goasciinema/internal/config"
	"github.com/spf13/cobra"
)

var version = "1.0.0"

// AppConfig holds the loaded configuration
var AppConfig *config.Config

var rootCmd = &cobra.Command{
	Use:   "goasciinema",
	Short: "Record and share terminal sessions",
	Long: `goasciinema - A fast terminal session recorder written in Go.

Record terminal sessions and share them on asciinema.org or locally.
This is a Go implementation of asciinema, optimized for performance.

Configuration:
  Create ~/.goasciinema with:
    database = ~/console-logs/asciinema_logs.db

  Or set GOASCIINEMA_DATABASE environment variable.`,
	Version: version,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// GetDefaultDatabasePath returns the configured default database path
func GetDefaultDatabasePath() string {
	if AppConfig != nil {
		return AppConfig.GetDatabasePath()
	}
	return "asciinema_logs.db"
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func initConfig() {
	var err error
	AppConfig, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
	}
}
