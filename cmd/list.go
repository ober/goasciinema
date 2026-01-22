package cmd

import (
	"fmt"

	"github.com/ober/goasciinema/internal/database"
	"github.com/spf13/cobra"
)

var listDatabase string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List processed sessions",
	Long:  `List all processed asciinema sessions stored in the database.`,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listDatabase, "database", "d", "", "SQLite database file (default: from ~/.goasciinema or ~/console-logs/asciinema_logs.db)")
}

func runList(cmd *cobra.Command, args []string) error {
	// Use config default if no database specified
	dbPath := listDatabase
	if dbPath == "" {
		dbPath = GetDefaultDatabasePath()
	}

	// Open database
	db, err := database.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	sessions, err := db.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found. Run 'process' first.")
		return nil
	}

	// Print header
	fmt.Printf("%-35s %-20s %-10s %-10s\n", "Filename", "Session Date", "Size", "Chars")
	fmt.Println(repeatString("=", 80))

	for _, s := range sessions {
		fmt.Printf("%-35s %-20s %-10s %-10d\n",
			truncateString(s.Filename, 35),
			s.SessionDate,
			s.Dimensions,
			s.ContentSize,
		)
	}

	return nil
}

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
