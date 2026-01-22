package cmd

import (
	"fmt"

	"github.com/ober/goasciinema/internal/database"
	"github.com/spf13/cobra"
)

var statsDatabase string

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show database statistics",
	Long:  `Display statistics about the processed asciinema recordings database.`,
	RunE:  runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().StringVarP(&statsDatabase, "database", "d", "", "SQLite database file (default: from ~/.goasciinema or ~/console-logs/asciinema_logs.db)")
}

func runStats(cmd *cobra.Command, args []string) error {
	// Use config default if no database specified
	dbPath := statsDatabase
	if dbPath == "" {
		dbPath = GetDefaultDatabasePath()
	}

	// Open database
	db, err := database.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	stats, err := db.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("Database: %s\n", dbPath)
	fmt.Printf("Processed files: %d\n", stats.ProcessedFiles)
	fmt.Printf("Sessions: %d\n", stats.Sessions)
	fmt.Printf("Total characters: %s\n", formatNumber(stats.TotalChars))

	return nil
}

// formatNumber adds comma separators to large numbers
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	str := fmt.Sprintf("%d", n)
	var result []byte
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
