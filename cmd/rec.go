package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/ober/goasciinema/internal/config"
	"github.com/ober/goasciinema/internal/recorder"
	"github.com/spf13/cobra"
)

var recCmd = &cobra.Command{
	Use:   "rec [filename]",
	Short: "Record terminal session",
	Long: `Record a terminal session to a file.

If no filename is specified, a temporary file will be used.
The recording will be saved in asciicast v2 format.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRec,
}

var (
	recStdin         bool
	recAppend        bool
	recCommand       string
	recTitle         string
	recIdleTimeLimit float64
	recCols          int
	recRows          int
	recQuiet         bool
	recOverwrite     bool
)

func init() {
	rootCmd.AddCommand(recCmd)

	recCmd.Flags().BoolVar(&recStdin, "stdin", false, "Enable stdin recording")
	recCmd.Flags().BoolVar(&recAppend, "append", false, "Append to existing recording")
	recCmd.Flags().StringVarP(&recCommand, "command", "c", "", "Command to record (default: $SHELL)")
	recCmd.Flags().StringVarP(&recTitle, "title", "t", "", "Title of the recording")
	recCmd.Flags().Float64VarP(&recIdleTimeLimit, "idle-time-limit", "i", 0, "Limit recorded idle time to given seconds")
	recCmd.Flags().IntVar(&recCols, "cols", 0, "Override terminal columns")
	recCmd.Flags().IntVar(&recRows, "rows", 0, "Override terminal rows")
	recCmd.Flags().BoolVarP(&recQuiet, "quiet", "q", false, "Quiet mode (suppress notices)")
	recCmd.Flags().BoolVarP(&recOverwrite, "overwrite", "y", false, "Overwrite existing file without asking")
}

func runRec(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine filename
	var filename string
	if len(args) > 0 {
		filename = args[0]
	} else {
		// Generate temporary filename
		filename = fmt.Sprintf("/tmp/goasciinema-%d.cast", time.Now().Unix())
	}

	// Check if file exists
	if !recAppend && !recOverwrite {
		if _, err := os.Stat(filename); err == nil {
			fmt.Fprintf(os.Stderr, "File %s already exists. Use --overwrite to overwrite.\n", filename)
			return nil
		}
	}

	// Apply config defaults
	if recCommand == "" {
		recCommand = cfg.Record.Command
	}
	if recIdleTimeLimit == 0 {
		recIdleTimeLimit = cfg.Record.IdleTimeLimit
	}
	if !recStdin {
		recStdin = cfg.Record.Stdin
	}

	if !recQuiet && !cfg.Record.Quiet {
		fmt.Fprintf(os.Stderr, "Recording terminal session to %s\n", filename)
		fmt.Fprintf(os.Stderr, "Press Ctrl+D or type 'exit' to end recording.\n")
	}

	// Create recorder
	rec := recorder.New(recorder.Options{
		Command:       recCommand,
		Title:         recTitle,
		IdleTimeLimit: recIdleTimeLimit,
		RecordStdin:   recStdin,
		Append:        recAppend,
		Cols:          recCols,
		Rows:          recRows,
	})

	// Start recording
	err = rec.Record(filename)
	if err != nil {
		return fmt.Errorf("recording failed: %w", err)
	}

	if !recQuiet && !cfg.Record.Quiet {
		fmt.Fprintf(os.Stderr, "\nRecording finished. Saved to %s\n", filename)
	}

	return nil
}
