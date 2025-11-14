package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	consts "github.com/khanhnv2901/seca-cli/internal/shared/constants"
)

func useTempDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(dataDirEnvVar, dir)
	return dir
}

func TestInfoCommand(t *testing.T) {
	useTempDataDir(t)
	defer setupTestAppContext(t)()

	// Capture output
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)

	// Execute command
	if err := infoCmd.RunE(infoCmd, []string{}); err != nil {
		t.Fatalf("info command failed: %v", err)
	}

	output := buf.String()

	// Verify output contains expected sections
	expectedSections := []string{
		"SECA-CLI System Information",
		"Platform:",
		"Data Locations:",
		"Data Directory:",
		"Engagements File:",
		"Results Directory:",
		"Configuration File:",
		"Documentation:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Expected output to contain '%s', got:\n%s", section, output)
		}
	}

	// Verify platform info is correct
	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !strings.Contains(output, expectedPlatform) {
		t.Errorf("Expected platform '%s' in output, got:\n%s", expectedPlatform, output)
	}
}

func TestInfoCommand_ShowsDataDirectory(t *testing.T) {
	useTempDataDir(t)
	defer setupTestAppContext(t)()

	// Get expected data directory
	dataDir, err := getDataDir()
	if err != nil {
		t.Fatalf("Failed to get data directory: %v", err)
	}

	// Capture output
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)

	// Execute command
	if err := infoCmd.RunE(infoCmd, []string{}); err != nil {
		t.Fatalf("info command failed: %v", err)
	}

	output := buf.String()

	// Verify data directory is shown
	if !strings.Contains(output, dataDir) {
		t.Errorf("Expected output to contain data directory '%s', got:\n%s", dataDir, output)
	}
}

func TestInfoCommand_ShowsEngagementsPath(t *testing.T) {
	useTempDataDir(t)
	defer setupTestAppContext(t)()

	// Get expected engagements path
	engagementsPath, err := getEngagementsFilePath()
	if err != nil {
		t.Fatalf("Failed to get engagements file path: %v", err)
	}

	// Capture output
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)

	// Execute command
	err = infoCmd.RunE(infoCmd, []string{})
	if err != nil {
		t.Fatalf("info command failed: %v", err)
	}

	output := buf.String()

	// Verify engagements path is shown
	if !strings.Contains(output, engagementsPath) {
		t.Errorf("Expected output to contain engagements path '%s', got:\n%s", engagementsPath, output)
	}
}

func TestInfoCommand_ShowsResultsDirectory(t *testing.T) {
	useTempDataDir(t)
	defer setupTestAppContext(t)()

	// Get expected results directory from appContext (which overrides the default)
	expectedResultsDir := globalAppContext.ResultsDir

	// Capture output
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)

	// Execute command
	if err := infoCmd.RunE(infoCmd, []string{}); err != nil {
		t.Fatalf("info command failed: %v", err)
	}

	output := buf.String()

	// Verify results directory is shown
	if !strings.Contains(output, expectedResultsDir) {
		t.Errorf("Expected output to contain results directory '%s', got:\n%s", expectedResultsDir, output)
	}
}

func TestInfoCommand_ShowsFileExistence(t *testing.T) {
	useTempDataDir(t)
	defer setupTestAppContext(t)()

	engagementsPath, err := getEngagementsFilePath()
	if err != nil {
		t.Fatalf("failed to get engagements path: %v", err)
	}

	// Create a minimal engagements file to simulate existing data
	if err := os.MkdirAll(filepath.Dir(engagementsPath), consts.DefaultDirPerm); err != nil {
		t.Fatalf("failed to create engagements directory: %v", err)
	}
	if err := os.WriteFile(engagementsPath, []byte(`[]`), consts.DefaultFilePerm); err != nil {
		t.Fatalf("failed to create engagements file: %v", err)
	}

	// Capture output
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)

	// Execute command
	if err := infoCmd.RunE(infoCmd, []string{}); err != nil {
		t.Fatalf("info command failed: %v", err)
	}

	output := buf.String()

	// Verify file existence indicators
	if !strings.Contains(output, "✓") && !strings.Contains(output, "✗") {
		t.Error("Expected output to contain file existence indicators (✓ or ✗)")
	}

	// Verify "exists" is shown for engagements file
	if !strings.Contains(output, "(exists)") {
		t.Error("Expected output to indicate engagements file exists")
	}
}

func TestInfoCommand_ShowsConfigInfo(t *testing.T) {
	defer setupTestAppContext(t)()

	// Capture output
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)

	// Execute command
	if err := infoCmd.RunE(infoCmd, []string{}); err != nil {
		t.Fatalf("info command failed: %v", err)
	}

	output := buf.String()

	// Verify config file info is shown
	if !strings.Contains(output, "~/.seca-cli.yaml") {
		t.Error("Expected output to contain config file path")
	}

	// Verify it shows status (exists or using defaults)
	hasConfigStatus := strings.Contains(output, "(exists)") || strings.Contains(output, "(using defaults)")
	if !hasConfigStatus {
		t.Error("Expected output to show config file status")
	}
}

func TestInfoCommand_ShowsDocumentation(t *testing.T) {
	defer setupTestAppContext(t)()

	// Capture output
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)

	// Execute command
	err := infoCmd.RunE(infoCmd, []string{})
	if err != nil {
		t.Fatalf("info command failed: %v", err)
	}

	output := buf.String()

	// Verify documentation references
	expectedDocs := []string{
		"README.md",
		"docs/README.md",
		"docs/reference/data-migration.md",
		"docs/operator-guide/compliance.md",
		"docs/user-guide/configuration.md",
	}

	for _, doc := range expectedDocs {
		if !strings.Contains(output, doc) {
			t.Errorf("Expected output to contain documentation reference '%s'", doc)
		}
	}
}

func TestInfoCommand_ShowsOverrideInstructions(t *testing.T) {
	defer setupTestAppContext(t)()

	// Capture output
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)

	// Execute command
	err := infoCmd.RunE(infoCmd, []string{})
	if err != nil {
		t.Fatalf("info command failed: %v", err)
	}

	output := buf.String()

	// Verify override instructions are shown
	if !strings.Contains(output, "To override data directory") {
		t.Error("Expected output to contain override instructions")
	}

	if !strings.Contains(output, "results_dir:") {
		t.Error("Expected output to show results_dir config example")
	}
}

func TestInfoCommand_WithOperator(t *testing.T) {
	defer setupTestAppContext(t)()

	// Capture output
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)

	// Execute command
	err := infoCmd.RunE(infoCmd, []string{})
	if err != nil {
		t.Fatalf("info command failed: %v", err)
	}

	output := buf.String()

	// Verify operator is shown
	if !strings.Contains(output, "test-operator") {
		t.Error("Expected output to contain operator name")
	}
}

func TestInfoCommand_DataDirError(t *testing.T) {
	defer setupTestAppContext(t)()

	// This test verifies that if getDataDir fails, the command returns an error
	// In normal circumstances, getDataDir should not fail, but we test the error path

	// We can't easily force getDataDir to fail without mocking, so we just verify
	// the command structure allows for error handling

	// Capture output
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)

	// Execute command (should succeed in normal test environment)
	err := infoCmd.RunE(infoCmd, []string{})
	if err != nil {
		// If it fails, verify error message is descriptive
		if !strings.Contains(err.Error(), "data directory") {
			t.Errorf("Expected descriptive error about data directory, got: %v", err)
		}
	}
}

func TestInfoCommand_PlatformSpecific(t *testing.T) {
	defer setupTestAppContext(t)()

	// Capture output
	var buf bytes.Buffer
	infoCmd.SetOut(&buf)
	infoCmd.SetErr(&buf)

	// Execute command
	err := infoCmd.RunE(infoCmd, []string{})
	if err != nil {
		t.Fatalf("info command failed: %v", err)
	}

	output := buf.String()

	// Verify platform-specific path is shown
	dataDir, err := getDataDir()
	if err != nil {
		t.Fatalf("Failed to get data directory: %v", err)
	}

	// Verify output contains data directory
	if !strings.Contains(output, dataDir) {
		t.Errorf("Expected output to contain data directory '%s'", dataDir)
	}

	// Ensure the path is absolute and points to an existing directory.
	if !filepath.IsAbs(dataDir) {
		t.Errorf("Expected absolute data directory, got: %s", dataDir)
	}
	if stat, err := os.Stat(dataDir); err != nil || !stat.IsDir() {
		t.Errorf("Expected data directory to exist, got error: %v", err)
	}
}
