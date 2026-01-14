package cmd

import (
	"fmt"

	"github.com/ober/goasciinema/internal/api"
	"github.com/ober/goasciinema/internal/config"
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload <filename>",
	Short: "Upload recorded session to asciinema.org",
	Long: `Upload an asciicast recording to asciinema.org.

The recording will be available at the returned URL.
Use 'goasciinema auth' to link the recording to your account.`,
	Args: cobra.ExactArgs(1),
	RunE: runUpload,
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}

func runUpload(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	filename := args[0]

	installID, err := cfg.GetInstallID()
	if err != nil {
		return fmt.Errorf("failed to get install ID: %w", err)
	}

	client := api.NewClient(cfg.API.URL, installID)

	fmt.Printf("Uploading %s...\n", filename)

	resp, err := client.Upload(filename)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	if resp.URL != "" {
		fmt.Printf("\nView recording at:\n%s\n", resp.URL)
	}
	if resp.Message != "" {
		fmt.Println(resp.Message)
	}

	return nil
}
