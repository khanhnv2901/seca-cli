package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/khanhnv2901/seca-cli/cmd/testutil"
	consts "github.com/khanhnv2901/seca-cli/internal/shared/constants"
)

func setDataDirOverride(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(dataDirEnvVar, dir)
	t.Setenv("XDG_DATA_HOME", "")
	return dir
}

func clearDataDirEnv(t *testing.T) {
	t.Helper()
	t.Setenv(dataDirEnvVar, "")
	t.Setenv("XDG_DATA_HOME", "")
}

func TestGetDataDir_DefaultLocation(t *testing.T) {
	clearDataDirEnv(t)
	baseHome := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		localAppData := filepath.Join(baseHome, "LocalAppData")
		appData := filepath.Join(baseHome, "AppData")
		if err := os.MkdirAll(localAppData, consts.DefaultDirPerm); err != nil {
			t.Fatalf("failed to create local app data dir: %v", err)
		}
		t.Setenv("LOCALAPPDATA", localAppData)
		t.Setenv("APPDATA", appData)
	default:
		t.Setenv("HOME", baseHome)
		t.Setenv("USERPROFILE", baseHome)
	}

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

	// Verify OS-specific path rooted in the temporary home
	switch runtime.GOOS {
	case "windows":
		expected := filepath.Join(os.Getenv("LOCALAPPDATA"), "seca-cli")
		if dataDir != expected {
			t.Errorf("Windows: expected %s, got %s", expected, dataDir)
		}
	case "darwin":
		expected := filepath.Join(baseHome, "Library", "Application Support", "seca-cli")
		if dataDir != expected {
			t.Errorf("macOS: expected %s, got %s", expected, dataDir)
		}
	default: // Linux/Unix
		expected := filepath.Join(baseHome, ".local", "share", "seca-cli")
		if dataDir != expected {
			t.Errorf("Linux: expected %s, got %s", expected, dataDir)
		}
	}
}

func TestGetDataDir_Override(t *testing.T) {
	dir := setDataDirOverride(t)
	dataDir, err := getDataDir()
	if err != nil {
		t.Fatalf("getDataDir() override failed: %v", err)
	}
	if dataDir != dir {
		t.Fatalf("expected override dir %s, got %s", dir, dataDir)
	}
}

func TestGetEngagementsFilePath(t *testing.T) {
	override := setDataDirOverride(t)
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
	parentDir := filepath.Dir(filePath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Errorf("Parent directory does not exist: %s", parentDir)
	}

	if !strings.HasPrefix(filePath, override) {
		t.Errorf("Expected file path to be inside override dir %s, got %s", override, filePath)
	}
}

func TestGetResultsDir(t *testing.T) {
	setDataDirOverride(t)
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
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	oldPath := filepath.Join(env.TmpDir, "old_engagements.json")
	newPath := filepath.Join(env.TmpDir, "new_engagements.json")

	// Create old file with test data
	testData := []byte(`[{"id":"123","name":"Test"}]`)
	env.CreateFile("old_engagements.json", testData)

	// Migrate
	if err := migrateEngagementsFile(oldPath, newPath); err != nil {
		t.Fatalf("migrateEngagementsFile() failed: %v", err)
	}

	// Verify new file exists
	env.MustExist("new_engagements.json")

	// Verify content
	newData := env.ReadFile("new_engagements.json")
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
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	oldPath := filepath.Join(env.TmpDir, "nonexistent.json")
	newPath := filepath.Join(env.TmpDir, "new.json")

	// Try to migrate non-existent file
	err := migrateEngagementsFile(oldPath, newPath)
	if err == nil {
		t.Error("Expected error when migrating non-existent file")
	}
}

func TestGetEngagementsFilePath_Migration(t *testing.T) {
	setDataDirOverride(t)
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
	setDataDirOverride(t)
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
	if err := os.WriteFile(testFile, []byte("test"), consts.DefaultFilePerm); err != nil {
		t.Errorf("Cannot write to data directory: %v", err)
	} else {
		_ = os.Remove(testFile) // Clean up
	}
}

func TestGetDataDir_PermissionDenied(t *testing.T) {
	clearDataDirEnv(t)
	if runtime.GOOS == "windows" {
		t.Skip("permission bits behave differently on Windows")
	}

	tmp := t.TempDir()
	readOnly := filepath.Join(tmp, "readonly")
	if err := os.Mkdir(readOnly, 0o500); err != nil {
		t.Fatalf("failed to create readonly dir: %v", err)
	}

	t.Setenv("XDG_DATA_HOME", readOnly)

	if _, err := getDataDir(); err == nil {
		t.Fatal("expected error when creating data dir under read-only parent")
	}
}

func TestGetResultsDir_PermissionDenied(t *testing.T) {
	clearDataDirEnv(t)
	if runtime.GOOS == "windows" {
		t.Skip("permission bits behave differently on Windows")
	}

	tmp := t.TempDir()
	readOnly := filepath.Join(tmp, "readonly")
	if err := os.Mkdir(readOnly, 0o500); err != nil {
		t.Fatalf("failed to create readonly dir: %v", err)
	}

	t.Setenv("XDG_DATA_HOME", readOnly)

	if _, err := getResultsDir(); err == nil {
		t.Fatal("expected error when creating results dir under read-only parent")
	}
}
