package cmd

import (
	"fmt"
	"time"

	"github.com/ober/goasciinema/internal/database"
	"github.com/spf13/cobra"
)

var (
	searchContext  int
	searchLimit    int
	searchDatabase string
)

var searchCmd = &cobra.Command{
	Use:   "search <term>",
	Short: "Search for commands in the database",
	Long: `Search for a term in processed asciinema recordings.

Returns matching lines with surrounding context, formatted in org-mode style.
The search is case-insensitive.`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().IntVarP(&searchContext, "context", "c", 5, "Number of context lines before/after match")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 50, "Maximum number of results")
	searchCmd.Flags().StringVarP(&searchDatabase, "database", "d", "asciinema_logs.db", "SQLite database file")
}

func runSearch(cmd *cobra.Command, args []string) error {
	term := args[0]

	// Open database
	db, err := database.Open(searchDatabase)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	results, err := db.Search(term, searchContext, searchLimit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Printf("# No matches found for: %s\n", term)
		return nil
	}

	// Org-mode header
	fmt.Printf("#+TITLE: Search Results for \"%s\"\n", term)
	fmt.Printf("#+DATE: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("#+RESULTS: %d match(es)\n", len(results))
	fmt.Println()

	for i, result := range results {
		fmt.Printf("* Match %d: %s\n", i+1, result.Filename)
		fmt.Println(":PROPERTIES:")
		fmt.Printf(":SESSION_DATE: %s\n", result.SessionDate)
		fmt.Printf(":LINE_NUMBER: %d\n", result.LineNumber)
		// Truncate matched text to 80 chars
		matchedText := result.MatchedText
		if len(matchedText) > 80 {
			matchedText = matchedText[:80]
		}
		fmt.Printf(":MATCHED_TEXT: %s\n", matchedText)
		fmt.Println(":END:")
		fmt.Println()
		fmt.Println("#+begin_src shell")
		fmt.Println(result.Context)
		fmt.Println("#+end_src")
		fmt.Println()
	}

	return nil
}
