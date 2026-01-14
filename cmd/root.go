package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "1.0.0"

var rootCmd = &cobra.Command{
	Use:   "goasciinema",
	Short: "Record and share terminal sessions",
	Long: `goasciinema - A fast terminal session recorder written in Go.

Record terminal sessions and share them on asciinema.org or locally.
This is a Go implementation of asciinema, optimized for performance.`,
	Version: version,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
