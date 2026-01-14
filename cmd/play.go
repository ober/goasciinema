package cmd

import (
	"fmt"

	"github.com/ober/goasciinema/internal/config"
	"github.com/ober/goasciinema/internal/player"
	"github.com/spf13/cobra"
)

var playCmd = &cobra.Command{
	Use:   "play <filename>",
	Short: "Replay recorded terminal session",
	Long: `Play back a recorded asciicast file.

Supports both local files and URLs.
Use -s to adjust playback speed, -i to limit idle time.`,
	Args: cobra.ExactArgs(1),
	RunE: runPlay,
}

var (
	playSpeed         float64
	playIdleTimeLimit float64
	playMaxWait       float64
	playLoop          bool
)

func init() {
	rootCmd.AddCommand(playCmd)

	playCmd.Flags().Float64VarP(&playSpeed, "speed", "s", 1.0, "Playback speed (e.g., 2 for 2x speed)")
	playCmd.Flags().Float64VarP(&playIdleTimeLimit, "idle-time-limit", "i", 0, "Limit replayed idle time to given seconds")
	playCmd.Flags().Float64VarP(&playMaxWait, "maxwait", "m", 0, "Maximum wait time between frames")
	playCmd.Flags().BoolVarP(&playLoop, "loop", "l", false, "Loop playback")
}

func runPlay(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	filename := args[0]

	// Apply config defaults
	if playSpeed == 1.0 && cfg.Play.Speed > 0 {
		playSpeed = cfg.Play.Speed
	}
	if playIdleTimeLimit == 0 {
		playIdleTimeLimit = cfg.Play.IdleTimeLimit
	}
	if playMaxWait == 0 {
		playMaxWait = cfg.Play.MaxWait
	}

	// Create player
	p := player.New(player.Options{
		Speed:         playSpeed,
		IdleTimeLimit: playIdleTimeLimit,
		MaxWait:       playMaxWait,
		Loop:          playLoop,
	})

	// Play
	err = p.Play(filename)
	if err != nil {
		return fmt.Errorf("playback failed: %w", err)
	}

	return nil
}
