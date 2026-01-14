package database

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// ProcessedFile represents a processed asciinema file
type ProcessedFile struct {
	ID          int64
	Filename    string
	Filepath    string
	FileHash    string
	ProcessedAt time.Time
}

// Session represents a session record in the database
type Session struct {
	ID        int64
	FileID    int64
	Version   int
	Width     int
	Height    int
	Timestamp int64
	Shell     string
	Term      string
	Content   string
}

// SessionInfo combines session and file info for listing
type SessionInfo struct {
	Filename    string
	SessionDate string
	Dimensions  string
	Shell       string
	ContentSize int
	ProcessedAt string
}

// SearchResult represents a search match with context
type SearchResult struct {
	Filename    string
	SessionDate string
	LineNumber  int
	MatchedText string
	Context     string
}

// Stats represents database statistics
type Stats struct {
	ProcessedFiles int
	Sessions       int
	TotalChars     int64
}

// Open opens or creates a SQLite database
func Open(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.init(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

// init creates the database schema
func (db *DB) init() error {
	// Enable foreign keys
	if _, err := db.conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create processed_files table
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS processed_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT UNIQUE NOT NULL,
			filepath TEXT NOT NULL,
			file_hash TEXT NOT NULL,
			processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create processed_files table: %w", err)
	}

	// Create sessions table
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_id INTEGER NOT NULL,
			version INTEGER,
			width INTEGER,
			height INTEGER,
			timestamp INTEGER,
			shell TEXT,
			term TEXT,
			content TEXT,
			FOREIGN KEY (file_id) REFERENCES processed_files(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	// Create indexes
	_, err = db.conn.Exec(`
		CREATE INDEX IF NOT EXISTS idx_processed_files_filename ON processed_files(filename);
		CREATE INDEX IF NOT EXISTS idx_sessions_file_id ON sessions(file_id);
	`)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// IsFileProcessed checks if a file has already been processed (and unchanged)
func (db *DB) IsFileProcessed(filepath string) (bool, error) {
	filename := getFilename(filepath)

	var storedHash string
	err := db.conn.QueryRow(
		"SELECT file_hash FROM processed_files WHERE filename = ?",
		filename,
	).Scan(&storedHash)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to query processed files: %w", err)
	}

	// Check if file has changed
	currentHash, err := fileHash(filepath)
	if err != nil {
		return false, err
	}

	return storedHash == currentHash, nil
}

// InsertFile inserts or updates a processed file and its session
func (db *DB) InsertFile(filepath string, header Header, content string) error {
	filename := getFilename(filepath)
	hash, err := fileHash(filepath)
	if err != nil {
		return err
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing record if present
	var existingID int64
	err = tx.QueryRow("SELECT id FROM processed_files WHERE filename = ?", filename).Scan(&existingID)
	if err == nil {
		_, err = tx.Exec("DELETE FROM processed_files WHERE id = ?", existingID)
		if err != nil {
			return fmt.Errorf("failed to delete existing record: %w", err)
		}
	}

	// Insert processed file
	result, err := tx.Exec(
		"INSERT INTO processed_files (filename, filepath, file_hash) VALUES (?, ?, ?)",
		filename, filepath, hash,
	)
	if err != nil {
		return fmt.Errorf("failed to insert processed file: %w", err)
	}

	fileID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	// Insert session
	_, err = tx.Exec(`
		INSERT INTO sessions (file_id, version, width, height, timestamp, shell, term, content)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, fileID, header.Version, header.Width, header.Height, header.Timestamp, header.Shell, header.Term, content)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}

	return tx.Commit()
}

// Search searches for a term in the database and returns matches with context
func (db *DB) Search(term string, contextLines, limit int) ([]SearchResult, error) {
	rows, err := db.conn.Query(`
		SELECT s.id, s.timestamp, s.content, p.filename
		FROM sessions s
		JOIN processed_files p ON s.file_id = p.id
		WHERE s.content LIKE ?
		ORDER BY p.filename
	`, "%"+term+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	termLower := strings.ToLower(term)

	for rows.Next() {
		var sessionID int64
		var timestamp sql.NullInt64
		var content, filename string

		if err := rows.Scan(&sessionID, &timestamp, &content, &filename); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		lines := strings.Split(content, "\n")

		for lineNum, line := range lines {
			if strings.Contains(strings.ToLower(line), termLower) {
				if len(results) >= limit {
					break
				}

				// Get context lines
				start := lineNum - contextLines
				if start < 0 {
					start = 0
				}
				end := lineNum + contextLines + 1
				if end > len(lines) {
					end = len(lines)
				}

				var snippetLines []string
				for i := start; i < end; i++ {
					if strings.TrimSpace(lines[i]) != "" {
						prefix := "    "
						if i == lineNum {
							prefix = ">>> "
						}
						snippetLines = append(snippetLines, prefix+lines[i])
					}
				}

				sessionDate := "Unknown"
				if timestamp.Valid {
					sessionDate = time.Unix(timestamp.Int64, 0).Format("2006-01-02 15:04:05")
				}

				results = append(results, SearchResult{
					Filename:    filename,
					SessionDate: sessionDate,
					LineNumber:  lineNum + 1,
					MatchedText: strings.TrimSpace(line),
					Context:     strings.Join(snippetLines, "\n"),
				})
			}
		}

		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

// ListSessions returns all processed sessions
func (db *DB) ListSessions() ([]SessionInfo, error) {
	rows, err := db.conn.Query(`
		SELECT p.filename, p.processed_at, s.timestamp, s.width, s.height, s.shell,
			   LENGTH(s.content) as content_size
		FROM processed_files p
		JOIN sessions s ON s.file_id = p.id
		ORDER BY p.filename
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var results []SessionInfo

	for rows.Next() {
		var filename, processedAt string
		var timestamp sql.NullInt64
		var width, height sql.NullInt64
		var shell sql.NullString
		var contentSize int

		if err := rows.Scan(&filename, &processedAt, &timestamp, &width, &height, &shell, &contentSize); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		sessionDate := "Unknown"
		if timestamp.Valid {
			sessionDate = time.Unix(timestamp.Int64, 0).Format("2006-01-02 15:04:05")
		}

		dimensions := "Unknown"
		if width.Valid && height.Valid {
			dimensions = fmt.Sprintf("%dx%d", width.Int64, height.Int64)
		}

		shellStr := "Unknown"
		if shell.Valid && shell.String != "" {
			shellStr = shell.String
		}

		results = append(results, SessionInfo{
			Filename:    filename,
			SessionDate: sessionDate,
			Dimensions:  dimensions,
			Shell:       shellStr,
			ContentSize: contentSize,
			ProcessedAt: processedAt,
		})
	}

	return results, nil
}

// GetStats returns database statistics
func (db *DB) GetStats() (*Stats, error) {
	var stats Stats

	err := db.conn.QueryRow("SELECT COUNT(*) FROM processed_files").Scan(&stats.ProcessedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to count processed files: %w", err)
	}

	err = db.conn.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&stats.Sessions)
	if err != nil {
		return nil, fmt.Errorf("failed to count sessions: %w", err)
	}

	var totalChars sql.NullInt64
	err = db.conn.QueryRow("SELECT SUM(LENGTH(content)) FROM sessions").Scan(&totalChars)
	if err != nil {
		return nil, fmt.Errorf("failed to sum content length: %w", err)
	}
	if totalChars.Valid {
		stats.TotalChars = totalChars.Int64
	}

	return &stats, nil
}

// Header contains asciinema header metadata for database storage
type Header struct {
	Version   int
	Width     int
	Height    int
	Timestamp int64
	Shell     string
	Term      string
}

// Helper functions

func getFilename(path string) string {
	return filepath.Base(path)
}

func fileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer file.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
