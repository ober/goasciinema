package cmd

import (
	"fmt"

	"github.com/ober/goasciinema/internal/player"
	"github.com/spf13/cobra"
)

var catCmd = &cobra.Command{
	Use:   "cat <filename>",
	Short: "Print full output of recorded session",
	Long: `Print the full output of an asciicast recording.

This outputs all the terminal output without any timing,
useful for extracting the raw content of a recording.`,
	Args: cobra.ExactArgs(1),
	RunE: runCat,
}

func init() {
	rootCmd.AddCommand(catCmd)
}

func runCat(cmd *cobra.Command, args []string) error {
	filename := args[0]

	err := player.Cat(filename)
	if err != nil {
		return fmt.Errorf("cat failed: %w", err)
	}

	return nil
}
