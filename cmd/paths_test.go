package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetDataDir(t *testing.T) {
	dataDir, err := getDataDir()
	if err != nil {
		t.Fatalf("getDataDir() failed: %v", err)
	}

	if dataDir == "" {
		t.Error("Expected non-empty data directory")
	}

	// Verify directory was created
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Errorf("Data directory was not created: %s", dataDir)
	}

	// Verify it contains "seca-cli"
	if !strings.Contains(dataDir, "seca-cli") {
		t.Errorf("Expected data directory to contain 'seca-cli', got: %s", dataDir)
	}

	// Verify OS-specific path
	switch runtime.GOOS {
	case "windows":
		if !strings.Contains(dataDir, "seca-cli") {
			t.Errorf("Windows: Expected path to contain seca-cli, got: %s", dataDir)
		}
	case "darwin":
		if !strings.Contains(dataDir, "Library") {
			t.Errorf("macOS: Expected path to contain Library, got: %s", dataDir)
		}
	default: // Linux/Unix
		homeDir, _ := os.UserHomeDir()
		expectedPrefix := filepath.Join(homeDir, ".local", "share")
		if !strings.HasPrefix(dataDir, expectedPrefix) {
			t.Errorf("Linux: Expected path to start with %s, got: %s", expectedPrefix, dataDir)
		}
	}
}

func TestGetEngagementsFilePath(t *testing.T) {
	filePath, err := getEngagementsFilePath()
	if err != nil {
		t.Fatalf("getEngagementsFilePath() failed: %v", err)
	}

	if filePath == "" {
		t.Error("Expected non-empty engagements file path")
	}

	if !strings.HasSuffix(filePath, "engagements.json") {
		t.Errorf("Expected path to end with engagements.json, got: %s", filePath)
	}

	// Verify parent directory exists
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("Parent directory does not exist: %s", dir)
	}
}

func TestGetResultsDir(t *testing.T) {
	resultsDir, err := getResultsDir()
	if err != nil {
		t.Fatalf("getResultsDir() failed: %v", err)
	}

	if resultsDir == "" {
		t.Error("Expected non-empty results directory")
	}

	// Verify directory was created
	if _, err := os.Stat(resultsDir); os.IsNotExist(err) {
		t.Errorf("Results directory was not created: %s", resultsDir)
	}

	if !strings.HasSuffix(resultsDir, "results") {
		t.Errorf("Expected path to end with results, got: %s", resultsDir)
	}
}

func TestMigrateEngagementsFile(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	oldPath := filepath.Join(tmpDir, "old_engagements.json")
	newPath := filepath.Join(tmpDir, "new_engagements.json")

	// Create old file with test data
	testData := []byte(`[{"id":"123","name":"Test"}]`)
	if err := os.WriteFile(oldPath, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Migrate
	if err := migrateEngagementsFile(oldPath, newPath); err != nil {
		t.Fatalf("migrateEngagementsFile() failed: %v", err)
	}

	// Verify new file exists
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("New file was not created")
	}

	// Verify content
	newData, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("Failed to read new file: %v", err)
	}

	if string(newData) != string(testData) {
		t.Errorf("Data mismatch: expected %s, got %s", testData, newData)
	}

	// Verify old file was backed up
	backupPath := oldPath + ".backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		// Old file might be deleted if backup fails, that's ok
		if _, err := os.Stat(oldPath); err == nil {
			t.Error("Old file should have been removed or backed up")
		}
	}
}

func TestMigrateEngagementsFile_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	oldPath := filepath.Join(tmpDir, "nonexistent.json")
	newPath := filepath.Join(tmpDir, "new.json")

	// Try to migrate non-existent file
	err := migrateEngagementsFile(oldPath, newPath)
	if err == nil {
		t.Error("Expected error when migrating non-existent file")
	}
}

func TestGetEngagementsFilePath_Migration(t *testing.T) {
	// This test is complex because it would need to mock the current directory
	// For now, we just verify the function doesn't error
	filePath, err := getEngagementsFilePath()
	if err != nil {
		t.Fatalf("getEngagementsFilePath() failed: %v", err)
	}

	if filePath == "" {
		t.Error("Expected non-empty file path")
	}
}

func TestDataDirCreation(t *testing.T) {
	// Get data dir (which creates it)
	dataDir, err := getDataDir()
	if err != nil {
		t.Fatalf("getDataDir() failed: %v", err)
	}

	// Verify it exists and is a directory
	info, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf("Data directory does not exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("Data directory path is not a directory")
	}

	// Verify permissions (should be readable/writable)
	testFile := filepath.Join(dataDir, "test_write.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Errorf("Cannot write to data directory: %v", err)
	} else {
		_ = os.Remove(testFile) // Clean up
	}
}
