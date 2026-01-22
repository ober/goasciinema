package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Config holds all configuration
type Config struct {
	API      APIConfig
	Record   RecordConfig
	Play     PlayConfig
	Database DatabaseConfig
	homeDir  string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path string
}

// APIConfig holds API-related configuration
type APIConfig struct {
	URL string
}

// RecordConfig holds recording configuration
type RecordConfig struct {
	Command       string
	Stdin         bool
	Env           []string
	IdleTimeLimit float64
	Quiet         bool
}

// PlayConfig holds playback configuration
type PlayConfig struct {
	Speed         float64
	IdleTimeLimit float64
	MaxWait       float64
}

// Load loads configuration from files and environment
func Load() (*Config, error) {
	home, _ := os.UserHomeDir()

	cfg := &Config{
		API: APIConfig{
			URL: "https://asciinema.org",
		},
		Record: RecordConfig{
			Env: []string{"SHELL", "TERM"},
		},
		Play: PlayConfig{
			Speed: 1.0,
		},
		Database: DatabaseConfig{
			Path: filepath.Join(home, "console-logs", "asciinema_logs.db"),
		},
	}

	// Get config directory
	configDir := getConfigDir()
	cfg.homeDir = configDir

	// Ensure config directory exists
	os.MkdirAll(configDir, 0755)

	// Load ~/.goasciinema config file first (simple key=value format)
	goasciinemaConfig := filepath.Join(home, ".goasciinema")
	if data, err := os.ReadFile(goasciinemaConfig); err == nil {
		parseGoasciinemaConfig(string(data), cfg)
	}

	// Load asciinema config file if exists (INI format)
	configFile := filepath.Join(configDir, "config")
	if data, err := os.ReadFile(configFile); err == nil {
		parseConfig(string(data), cfg)
	}

	// Override with environment variables
	if url := os.Getenv("ASCIINEMA_API_URL"); url != "" {
		cfg.API.URL = url
	}
	if dbPath := os.Getenv("GOASCIINEMA_DATABASE"); dbPath != "" {
		cfg.Database.Path = expandPath(dbPath)
	}

	return cfg, nil
}

// GetDatabasePath returns the configured database path
func (c *Config) GetDatabasePath() string {
	return c.Database.Path
}

// parseGoasciinemaConfig parses the simple ~/.goasciinema config file
func parseGoasciinemaConfig(content string, cfg *Config) {
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Key-value pair
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "database":
			cfg.Database.Path = expandPath(value)
		}
	}
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// GetInstallID returns the install ID, creating one if necessary
func (c *Config) GetInstallID() (string, error) {
	idFile := filepath.Join(c.homeDir, "install-id")

	// Check environment variable first
	if id := os.Getenv("ASCIINEMA_INSTALL_ID"); id != "" {
		return id, nil
	}

	// Try to read existing ID
	if data, err := os.ReadFile(idFile); err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	// Generate new ID
	id := uuid.New().String()

	// Save ID
	os.WriteFile(idFile, []byte(id+"\n"), 0644)

	return id, nil
}

func getConfigDir() string {
	// Check ASCIINEMA_CONFIG_HOME first
	if dir := os.Getenv("ASCIINEMA_CONFIG_HOME"); dir != "" {
		return dir
	}

	// Check XDG_CONFIG_HOME
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "asciinema")
	}

	// Default to ~/.config/asciinema
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "asciinema")
}

func parseConfig(content string, cfg *Config) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.ToLower(line[1 : len(line)-1])
			continue
		}

		// Key-value pair
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch currentSection {
		case "api":
			switch key {
			case "url":
				cfg.API.URL = value
			}
		case "record":
			switch key {
			case "command":
				cfg.Record.Command = value
			case "stdin":
				cfg.Record.Stdin = value == "yes" || value == "true" || value == "1"
			case "idle_time_limit":
				cfg.Record.IdleTimeLimit, _ = strconv.ParseFloat(value, 64)
			case "quiet":
				cfg.Record.Quiet = value == "yes" || value == "true" || value == "1"
			}
		case "play":
			switch key {
			case "speed":
				cfg.Play.Speed, _ = strconv.ParseFloat(value, 64)
			case "idle_time_limit":
				cfg.Play.IdleTimeLimit, _ = strconv.ParseFloat(value, 64)
			case "maxwait":
				cfg.Play.MaxWait, _ = strconv.ParseFloat(value, 64)
			}
		}
	}
}
