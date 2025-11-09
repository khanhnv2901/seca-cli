package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// getDataDir returns the appropriate data directory for the current OS
// following XDG Base Directory specification on Linux/Unix
func getDataDir() (string, error) {
	var baseDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: %LOCALAPPDATA%\seca-cli
		baseDir = os.Getenv("LOCALAPPDATA")
		if baseDir == "" {
			baseDir = os.Getenv("APPDATA")
		}
		if baseDir == "" {
			return "", fmt.Errorf("could not determine Windows data directory")
		}
		baseDir = filepath.Join(baseDir, "seca-cli")

	case "darwin":
		// macOS: ~/Library/Application Support/seca-cli
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, "Library", "Application Support", "seca-cli")

	default:
		// Linux/Unix: Follow XDG Base Directory specification
		// Priority: $XDG_DATA_HOME/seca-cli > ~/.local/share/seca-cli
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			baseDir = filepath.Join(xdgDataHome, "seca-cli")
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("could not determine home directory: %w", err)
			}
			baseDir = filepath.Join(homeDir, ".local", "share", "seca-cli")
		}
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return baseDir, nil
}

// getEngagementsFilePath returns the path to engagements.json
// with automatic migration from old location if needed
func getEngagementsFilePath() (string, error) {
	// Get new data directory path
	dataDir, err := getDataDir()
	if err != nil {
		return "", err
	}

	newPath := filepath.Join(dataDir, "engagements.json")

	// Check if file exists in new location
	if _, err := os.Stat(newPath); err == nil {
		return newPath, nil
	}

	// Check for old location (current directory)
	oldPath := "engagements.json"
	if _, err := os.Stat(oldPath); err == nil {
		// Migrate from old location to new location
		if err := migrateEngagementsFile(oldPath, newPath); err != nil {
			// If migration fails, log warning but continue with new path
			fmt.Fprintf(os.Stderr, "Warning: Could not migrate engagements.json: %v\n", err)
			fmt.Fprintf(os.Stderr, "Using new location: %s\n", newPath)
		} else {
			fmt.Fprintf(os.Stderr, "Migrated engagements.json from %s to %s\n", oldPath, newPath)
		}
	}

	return newPath, nil
}

// migrateEngagementsFile moves engagements.json from old to new location
func migrateEngagementsFile(oldPath, newPath string) error {
	// Read old file
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read old file: %w", err)
	}

	// Write to new location
	if err := os.WriteFile(newPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write to new location: %w", err)
	}

	// Backup old file instead of deleting
	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		// If rename fails, just delete the old file
		_ = os.Remove(oldPath)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Created backup: %s\n", backupPath)
	return nil
}

// getResultsDir returns the path to the results directory
func getResultsDir() (string, error) {
	dataDir, err := getDataDir()
	if err != nil {
		return "", err
	}

	resultsDir := filepath.Join(dataDir, "results")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create results directory: %w", err)
	}

	return resultsDir, nil
}
