package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ober/goasciinema/internal/asciicast"
	"github.com/ober/goasciinema/internal/database"
	"github.com/ober/goasciinema/internal/sanitize"
	"github.com/spf13/cobra"
)

var (
	processForce    bool
	processDatabase string
)

var processCmd = &cobra.Command{
	Use:   "process [path]",
	Short: "Process .asc/.cast files into SQLite database",
	Long: `Process asciinema recording files and store them in a SQLite database.

This command reads .asc or .cast files, extracts the terminal output,
strips ANSI escape codes, and stores the clean content in a searchable
SQLite database.

Files are tracked by hash - unchanged files will be skipped unless --force is used.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProcess,
}

func init() {
	rootCmd.AddCommand(processCmd)
	processCmd.Flags().BoolVarP(&processForce, "force", "f", false, "Force reprocessing of already processed files")
	processCmd.Flags().StringVarP(&processDatabase, "database", "d", "", "SQLite database file (default: from ~/.goasciinema or ~/console-logs/asciinema_logs.db)")
}

func runProcess(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Use config default if no database specified
	dbPath := processDatabase
	if dbPath == "" {
		dbPath = GetDefaultDatabasePath()
	}

	// Open database
	db, err := database.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path not found: %w", err)
	}

	if info.IsDir() {
		processed, skipped, err := processDirectory(db, path)
		if err != nil {
			return err
		}
		fmt.Printf("\nSummary: %d processed, %d skipped\n", processed, skipped)
	} else {
		wasProcessed, err := processFile(db, path)
		if err != nil {
			return err
		}
		if wasProcessed {
			fmt.Printf("Processed: %s\n", filepath.Base(path))
		} else {
			fmt.Printf("Skipped (already processed): %s\n", filepath.Base(path))
		}
	}

	return nil
}

func processDirectory(db *database.DB, dir string) (int, int, error) {
	var processed, skipped int

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read directory: %w", err)
	}

	// Sort and filter for .asc and .cast files
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".asc") || strings.HasSuffix(name, ".cast") {
			files = append(files, filepath.Join(dir, name))
		}
	}

	for _, file := range files {
		wasProcessed, err := processFile(db, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to process %s: %v\n", file, err)
			continue
		}
		if wasProcessed {
			processed++
			fmt.Printf("Processed: %s\n", filepath.Base(file))
		} else {
			skipped++
		}
	}

	return processed, skipped, nil
}

func processFile(db *database.DB, filepath string) (bool, error) {
	// Check if already processed (unless force)
	if !processForce {
		isProcessed, err := db.IsFileProcessed(filepath)
		if err != nil {
			return false, err
		}
		if isProcessed {
			return false, nil
		}
	}

	// Open and read the asciicast file
	reader, err := asciicast.Open(filepath)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	// Extract all output content
	var content strings.Builder
	for {
		event, err := reader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				break
			}
			return false, fmt.Errorf("failed to read event: %w", err)
		}

		if event.Type == asciicast.EventTypeOutput {
			content.WriteString(event.Data)
		}
	}

	// Strip ANSI codes
	cleanContent := sanitize.StripANSI(content.String())

	// Get header info for database
	header := database.Header{
		Version:   reader.Header.Version,
		Width:     reader.Header.Width,
		Height:    reader.Header.Height,
		Timestamp: reader.Header.Timestamp,
	}

	// Extract shell and term from env if present
	if reader.Header.Env != nil {
		header.Shell = reader.Header.Env["SHELL"]
		header.Term = reader.Header.Env["TERM"]
	}

	// Insert into database
	if err := db.InsertFile(filepath, header, cleanContent); err != nil {
		return false, fmt.Errorf("failed to insert into database: %w", err)
	}

	return true, nil
}
