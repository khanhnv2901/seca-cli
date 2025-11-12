package cmd

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendAuditRow(t *testing.T) {
	// Setup temporary results directory
	tmpDir := "test_results"
	engagementID := "test123"
	defer os.RemoveAll(tmpDir)

	// Create directory
	if err := os.MkdirAll(filepath.Join(tmpDir, engagementID), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Test data
	err := AppendAuditRow(
		tmpDir, // resultsDir parameter
		engagementID,
		"test-operator",
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

	// Verify file exists
	auditPath := filepath.Join(tmpDir, engagementID, "audit.csv")
	if _, err := os.Stat(auditPath); os.IsNotExist(err) {
		t.Fatal("Audit file was not created")
	}

	// Read and verify content
	file, err := os.Open(auditPath)
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
	if dataRow[1] != engagementID {
		t.Errorf("Expected engagement_id '%s', got '%s'", engagementID, dataRow[1])
	}
	if dataRow[2] != "test-operator" {
		t.Errorf("Expected operator 'test-operator', got '%s'", dataRow[2])
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
	tmpDir := "test_results_multiple"
	engagementID := "test456"
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, engagementID), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Append multiple rows
	for i := 0; i < 3; i++ {
		err := AppendAuditRow(
			tmpDir, // resultsDir parameter
			engagementID,
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
	auditPath := filepath.Join(tmpDir, engagementID, "audit.csv")
	file, _ := os.Open(auditPath)
	defer file.Close()

	reader := csv.NewReader(file)
	records, _ := reader.ReadAll()

	if len(records) != 4 {
		t.Fatalf("Expected 4 rows (header + 3 data), got %d", len(records))
	}
}

func TestAppendAuditRow_WithError(t *testing.T) {
	tmpDir := "test_results_error"
	engagementID := "test789"
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, engagementID), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Test with error
	err := AppendAuditRow(
		tmpDir, // resultsDir parameter
		engagementID,
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
	auditPath := filepath.Join(tmpDir, engagementID, "audit.csv")
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
	tmpDir := "test_results_raw"
	engagementID := "rawtest123"
	defer os.RemoveAll(tmpDir)

	// Test data
	headers := map[string][]string{
		"Content-Type":   {"text/html"},
		"Server":         {"nginx/1.21.0"},
		"Content-Length": {"1234"},
	}
	bodySnippet := "<html><body>Test content</body></html>"

	err := SaveRawCapture(tmpDir, engagementID, "https://example.com", headers, bodySnippet)
	if err != nil {
		t.Fatalf("SaveRawCapture failed: %v", err)
	}

	// Verify file was created
	dir := filepath.Join(tmpDir, engagementID)
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
	tmpDir := "test_hash"
	defer os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("This is a test file for SHA256 hashing")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

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
	hashFile := testFile + ".sha256"
	if _, err := os.Stat(hashFile); os.IsNotExist(err) {
		t.Fatal(".sha256 file was not created")
	}

	// Read and verify hash file content
	hashContent, err := os.ReadFile(hashFile)
	if err != nil {
		t.Fatalf("Failed to read hash file: %v", err)
	}

	hashStr := string(hashContent)
	if !strings.Contains(hashStr, hash) {
		t.Error("Hash not found in .sha256 file")
	}

	if !strings.Contains(hashStr, "test.txt") {
		t.Error("Filename not found in .sha256 file")
	}
}

func TestHashFileSHA256_NonExistentFile(t *testing.T) {
	_, err := HashFileSHA256("/non/existent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestHashFileSHA256_Consistency(t *testing.T) {
	tmpDir := "test_hash_consistency"
	defer os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	testFile := filepath.Join(tmpDir, "consistency.txt")
	testContent := []byte("Consistent content")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

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
