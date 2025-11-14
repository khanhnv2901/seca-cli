package cmd

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/khanhnv2901/seca-cli/cmd/testutil"
)

func TestAppendAuditRow(t *testing.T) {
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	// Test data
	err := AppendAuditRow(
		env.AppCtx.ResultsDir,
		env.EngagementID,
		env.Operator,
		"check http",
		"https://example.com",
		"ok",
		200,
		"2026-01-15T00:00:00Z",
		"test note",
		"",
		1.234,
	)

	if err != nil {
		t.Fatalf("AppendAuditRow failed: %v", err)
	}

	// Verify file exists using helper
	auditPath := filepath.Join(env.EngagementID, "audit.csv")
	env.MustExist(filepath.Join("results", auditPath))

	// Read and verify content
	fullPath := filepath.Join(env.AppCtx.ResultsDir, env.EngagementID, "audit.csv")
	file, err := os.Open(fullPath)
	if err != nil {
		t.Fatalf("Failed to open audit file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	// Should have header + 1 data row
	if len(records) != 2 {
		t.Fatalf("Expected 2 rows (header + data), got %d", len(records))
	}

	// Verify header
	expectedHeader := []string{
		"timestamp", "engagement_id", "operator", "command", "target",
		"status", "http_status", "tls_expiry", "notes", "error", "duration_seconds",
	}

	for i, col := range expectedHeader {
		if records[0][i] != col {
			t.Errorf("Header column %d: expected '%s', got '%s'", i, col, records[0][i])
		}
	}

	// Verify data row
	dataRow := records[1]
	if dataRow[1] != env.EngagementID {
		t.Errorf("Expected engagement_id '%s', got '%s'", env.EngagementID, dataRow[1])
	}
	if dataRow[2] != env.Operator {
		t.Errorf("Expected operator '%s', got '%s'", env.Operator, dataRow[2])
	}
	if dataRow[3] != "check http" {
		t.Errorf("Expected command 'check http', got '%s'", dataRow[3])
	}
	if dataRow[4] != "https://example.com" {
		t.Errorf("Expected target 'https://example.com', got '%s'", dataRow[4])
	}
	if dataRow[5] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", dataRow[5])
	}
	if dataRow[6] != "200" {
		t.Errorf("Expected http_status '200', got '%s'", dataRow[6])
	}
}

func TestAppendAuditRow_MultipleRows(t *testing.T) {
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	// Append multiple rows
	for i := 0; i < 3; i++ {
		err := AppendAuditRow(
			env.AppCtx.ResultsDir,
			env.EngagementID,
			"operator",
			"check http",
			"https://example.com",
			"ok",
			200,
			"",
			"",
			"",
			0.5,
		)
		if err != nil {
			t.Fatalf("AppendAuditRow %d failed: %v", i, err)
		}
	}

	// Verify we have header + 3 data rows
	auditPath := filepath.Join(env.AppCtx.ResultsDir, env.EngagementID, "audit.csv")
	file, _ := os.Open(auditPath)
	defer file.Close()

	reader := csv.NewReader(file)
	records, _ := reader.ReadAll()

	if len(records) != 4 {
		t.Fatalf("Expected 4 rows (header + 3 data), got %d", len(records))
	}
}

func TestAppendAuditRow_WithError(t *testing.T) {
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	// Test with error
	err := AppendAuditRow(
		env.AppCtx.ResultsDir,
		env.EngagementID,
		"operator",
		"check http",
		"https://example.com",
		"error",
		0,
		"",
		"",
		"connection timeout",
		2.5,
	)

	if err != nil {
		t.Fatalf("AppendAuditRow failed: %v", err)
	}

	// Verify error is recorded
	auditPath := filepath.Join(env.AppCtx.ResultsDir, env.EngagementID, "audit.csv")
	file, _ := os.Open(auditPath)
	defer file.Close()

	reader := csv.NewReader(file)
	records, _ := reader.ReadAll()

	if records[1][5] != "error" {
		t.Errorf("Expected status 'error', got '%s'", records[1][5])
	}
	if records[1][9] != "connection timeout" {
		t.Errorf("Expected error 'connection timeout', got '%s'", records[1][9])
	}
}

func TestSaveRawCapture(t *testing.T) {
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	// Test data
	headers := map[string][]string{
		"Content-Type":   {"text/html"},
		"Server":         {"nginx/1.21.0"},
		"Content-Length": {"1234"},
	}
	bodySnippet := "<html><body>Test content</body></html>"

	err := SaveRawCapture(env.AppCtx.ResultsDir, env.EngagementID, "https://example.com", headers, bodySnippet)
	if err != nil {
		t.Fatalf("SaveRawCapture failed: %v", err)
	}

	// Verify file was created
	dir := filepath.Join(env.AppCtx.ResultsDir, env.EngagementID)
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No raw capture file was created")
	}

	// Read the file and verify content
	var rawFile string
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "raw_") {
			rawFile = filepath.Join(dir, f.Name())
			break
		}
	}

	if rawFile == "" {
		t.Fatal("No raw_*.txt file found")
	}

	content, err := os.ReadFile(rawFile)
	if err != nil {
		t.Fatalf("Failed to read raw file: %v", err)
	}

	contentStr := string(content)

	// Verify target is in the file
	if !strings.Contains(contentStr, "https://example.com") {
		t.Error("Target URL not found in raw capture")
	}

	// Verify headers are in the file
	if !strings.Contains(contentStr, "Content-Type") {
		t.Error("Content-Type header not found")
	}

	// Verify body snippet is in the file
	if !strings.Contains(contentStr, "Test content") {
		t.Error("Body snippet not found in raw capture")
	}
}

func TestHashFileSHA256(t *testing.T) {
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test file using helper
	testContent := []byte("This is a test file for SHA256 hashing")
	testFile := env.CreateFile("test.txt", testContent)

	// Generate hash
	hash, err := HashFileSHA256(testFile)
	if err != nil {
		t.Fatalf("HashFileSHA256 failed: %v", err)
	}

	// Verify hash is not empty
	if hash == "" {
		t.Error("Hash is empty")
	}

	// Verify hash length (SHA256 is 64 hex characters)
	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}

	// Verify .sha256 file was created
	env.MustExist("test.txt.sha256")

	// Read and verify hash file content
	hashContent := env.ReadFile("test.txt.sha256")
	hashStr := string(hashContent)

	if !strings.Contains(hashStr, hash) {
		t.Error("Hash not found in .sha256 file")
	}

	if !strings.Contains(hashStr, "test.txt") {
		t.Error("Filename not found in .sha256 file")
	}
}

func TestHashFileSHA512(t *testing.T) {
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	testFile := env.CreateFile("sha512.txt", []byte("hash me with sha512"))

	hash, err := HashFile(testFile, HashAlgorithmSHA512)
	if err != nil {
		t.Fatalf("HashFile sha512 failed: %v", err)
	}

	if len(hash) != 128 {
		t.Errorf("Expected hash length 128 for sha512, got %d", len(hash))
	}

	env.MustExist("sha512.txt.sha512")
}

func TestHashFileSHA256_NonExistentFile(t *testing.T) {
	_, err := HashFileSHA256("/non/existent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestHashFileSHA256_Consistency(t *testing.T) {
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	// Create test file using helper
	testContent := []byte("Consistent content")
	testFile := env.CreateFile("consistency.txt", testContent)

	// Generate hash twice
	hash1, _ := HashFileSHA256(testFile)

	// Remove the .sha256 file and regenerate
	os.Remove(testFile + ".sha256")
	hash2, _ := HashFileSHA256(testFile)

	// Hashes should be identical
	if hash1 != hash2 {
		t.Errorf("Hashes are not consistent: '%s' != '%s'", hash1, hash2)
	}
}

func TestParseHashAlgorithm(t *testing.T) {
	tests := []struct {
		input    string
		expected HashAlgorithm
		hasError bool
	}{
		{"sha256", HashAlgorithmSHA256, false},
		{"SHA512", HashAlgorithmSHA512, false},
		{"", HashAlgorithmSHA256, false},
		{"md5", "", true},
	}

	for _, tt := range tests {
		algo, err := ParseHashAlgorithm(tt.input)
		if tt.hasError {
			if err == nil {
				t.Fatalf("expected error for %s", tt.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tt.input, err)
		}
		if algo != tt.expected {
			t.Fatalf("expected %s, got %s", tt.expected, algo)
		}
	}
}

func TestHashAlgorithmHelpers(t *testing.T) {
	tests := []struct {
		algo       HashAlgorithm
		display    string
		sumCommand string
	}{
		{HashAlgorithmSHA256, "SHA256", "sha256sum"},
		{HashAlgorithmSHA512, "SHA512", "sha512sum"},
		{"", "SHA256", "sha256sum"},
	}

	for _, tt := range tests {
		if got := tt.algo.DisplayName(); got != tt.display {
			t.Fatalf("DisplayName() = %s, want %s", got, tt.display)
		}
		if got := tt.algo.SumCommand(); got != tt.sumCommand {
			t.Fatalf("SumCommand() = %s, want %s", got, tt.sumCommand)
		}
	}
}
