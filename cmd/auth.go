package cmd

import (
	"fmt"

	"github.com/ober/goasciinema/internal/api"
	"github.com/ober/goasciinema/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage account authentication",
	Long: `Link this machine to your asciinema.org account.

Visit the URL shown to authenticate and link your recordings
to your account on asciinema.org.`,
	RunE: runAuth,
}

func init() {
	rootCmd.AddCommand(authCmd)
}

func runAuth(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	installID, err := cfg.GetInstallID()
	if err != nil {
		return fmt.Errorf("failed to get install ID: %w", err)
	}

	client := api.NewClient(cfg.API.URL, installID)

	fmt.Println("Open the following URL in a browser to link this machine")
	fmt.Println("to your asciinema.org account:")
	fmt.Println()
	fmt.Printf("    %s\n", client.AuthURL())
	fmt.Println()
	fmt.Println("This will associate all recordings uploaded from this machine")
	fmt.Println("(identified by your install ID) with your asciinema.org account,")
	fmt.Println("allowing you to manage them via the web interface.")
	fmt.Println()

	return nil
}
