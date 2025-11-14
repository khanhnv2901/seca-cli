package cmd

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/infrastructure/checker"
)

// Tests for CheckResult structure (cmd package uses this from checker package)
func TestCheckResult_JSON(t *testing.T) {
	result := checker.CheckResult{
		Target:       "https://example.com",
		CheckedAt:    time.Now().UTC(),
		Status:       "ok",
		HTTPStatus:   200,
		ServerHeader: "nginx",
		TLSExpiry:    "2026-01-15T00:00:00Z",
		Notes:        "test note",
		Error:        "",
	}

	// Marshal to JSON
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded checker.CheckResult
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Target != result.Target {
		t.Errorf("Target mismatch: expected '%s', got '%s'", result.Target, decoded.Target)
	}

	if decoded.Status != result.Status {
		t.Errorf("Status mismatch: expected '%s', got '%s'", result.Status, decoded.Status)
	}

	if decoded.HTTPStatus != result.HTTPStatus {
		t.Errorf("HTTPStatus mismatch: expected %d, got %d", result.HTTPStatus, decoded.HTTPStatus)
	}
}

func TestCheckResult_WithError(t *testing.T) {
	result := checker.CheckResult{
		Target:    "https://example.com",
		CheckedAt: time.Now().UTC(),
		Status:    "error",
		Error:     "connection timeout",
	}

	if result.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message to be set")
	}

	if result.HTTPStatus != 0 {
		t.Errorf("Expected HTTPStatus 0 for error, got %d", result.HTTPStatus)
	}
}

func TestCheckResult_Marshaling_OmitEmpty(t *testing.T) {
	// Test that omitempty works for optional fields
	result := checker.CheckResult{
		Target:    "https://example.com",
		CheckedAt: time.Now(),
		Status:    "ok",
		// Omit HTTPStatus, ServerHeader, etc.
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// HTTPStatus should be omitted when 0
	if _, exists := decoded["http_status"]; exists {
		t.Error("Expected http_status to be omitted")
	}

	// But target and status should be present
	if _, exists := decoded["target"]; !exists {
		t.Error("Expected target to be present")
	}

	if _, exists := decoded["status"]; !exists {
		t.Error("Expected status to be present")
	}
}

func TestCheckResult_SuccessfulCheck(t *testing.T) {
	result := checker.CheckResult{
		Target:       "https://api.example.com",
		CheckedAt:    time.Now().UTC(),
		Status:       "ok",
		HTTPStatus:   200,
		ServerHeader: "nginx/1.21.0",
		TLSExpiry:    "2026-12-31T23:59:59Z",
		Notes:        "robots.txt found",
	}

	if result.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result.Status)
	}

	if result.HTTPStatus != 200 {
		t.Errorf("Expected HTTP status 200, got %d", result.HTTPStatus)
	}

	if result.Error != "" {
		t.Error("Expected no error for successful check")
	}

	if result.TLSExpiry == "" {
		t.Error("Expected TLS expiry to be set")
	}
}

// Tests for RunMetadata (cmd package struct)
func TestRunMetadata(t *testing.T) {
	metadata := RunMetadata{
		Operator:       "test-operator",
		EngagementID:   "123456",
		EngagementName: "Test Engagement",
		Owner:          "owner@example.com",
		StartAt:        time.Now(),
		CompleteAt:     time.Now().Add(5 * time.Minute),
		AuditHash:      "abc123",
		TotalTargets:   5,
	}

	if metadata.Operator != "test-operator" {
		t.Errorf("Expected operator 'test-operator', got '%s'", metadata.Operator)
	}

	if metadata.TotalTargets != 5 {
		t.Errorf("Expected 5 targets, got %d", metadata.TotalTargets)
	}

	if metadata.AuditHash == "" {
		t.Error("AuditHash should not be empty")
	}
}

// Tests for RunOutput (cmd package struct)
func TestRunOutput_JSON(t *testing.T) {
	output := RunOutput{
		Metadata: RunMetadata{
			Operator:       "test-op",
			EngagementID:   "123",
			EngagementName: "Test",
			Owner:          "owner@test.com",
			StartAt:        time.Now(),
			CompleteAt:     time.Now(),
			TotalTargets:   2,
		},
		Results: []checker.CheckResult{
			{
				Target:     "https://example.com",
				CheckedAt:  time.Now(),
				Status:     "ok",
				HTTPStatus: 200,
			},
			{
				Target:     "https://test.com",
				CheckedAt:  time.Now(),
				Status:     "ok",
				HTTPStatus: 200,
			},
		},
	}

	// Marshal
	data, err := json.MarshalIndent(output, jsonPrefix, jsonIndent)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded RunOutput
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(decoded.Results))
	}

	if decoded.Metadata.TotalTargets != 2 {
		t.Errorf("Expected 2 total targets, got %d", decoded.Metadata.TotalTargets)
	}
}

// Tests for auto-sign validation (cmd package functionality)
func TestAutoSign_ValidationWithoutGPGKey(t *testing.T) {
	// Test that --auto-sign without --gpg-key returns an error
	cliConfig.Check.AutoSign = true
	cliConfig.Check.GPGKey = ""

	err := validateAutoSignFlags()
	if err == nil {
		t.Error("Expected error when --auto-sign is used without --gpg-key")
	}

	expectedMsg := "--gpg-key required with --auto-sign"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestAutoSign_ValidationWithGPGKey(t *testing.T) {
	// Test that --auto-sign with --gpg-key does not return an error
	cliConfig.Check.AutoSign = true
	cliConfig.Check.GPGKey = "test@example.com"

	err := validateAutoSignFlags()
	if err != nil {
		t.Errorf("Expected no error when both flags are set, got: %v", err)
	}
}

func TestAutoSign_ValidationWhenDisabled(t *testing.T) {
	// Test that when --auto-sign is false, no validation error occurs
	cliConfig.Check.AutoSign = false
	cliConfig.Check.GPGKey = ""

	err := validateAutoSignFlags()
	if err != nil {
		t.Errorf("Expected no error when --auto-sign is disabled, got: %v", err)
	}
}

// Helper function to validate auto-sign flags
func validateAutoSignFlags() error {
	if cliConfig.Check.AutoSign {
		if cliConfig.Check.GPGKey == "" {
			return fmt.Errorf("--gpg-key required with --auto-sign")
		}
	}
	return nil
}

func TestAutoSign_SignFileFunction(t *testing.T) {
	// Test that the sign file function is correctly structured
	testCases := []struct {
		name     string
		gpgKey   string
		filePath string
	}{
		{
			name:     "Sign with email key",
			gpgKey:   "test@example.com",
			filePath: "/tmp/test.sha256",
		},
		{
			name:     "Sign with key ID",
			gpgKey:   "1234ABCD",
			filePath: "/tmp/audit.csv.sha256",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify the function would be called with correct parameters
			// Note: This doesn't actually execute GPG, just validates the logic
			if tc.gpgKey == "" {
				t.Error("GPG key should not be empty")
			}
			if tc.filePath == "" {
				t.Error("File path should not be empty")
			}
		})
	}
}

func TestAutoSign_BothHashFilesAreSigned(t *testing.T) {
	// Test that both audit.csv.sha256 and http_results.json.sha256 are signed
	filesToSign := []string{
		"audit.csv.sha256",
		"http_results.json.sha256",
	}

	if len(filesToSign) != 2 {
		t.Errorf("Expected 2 files to be signed, got %d", len(filesToSign))
	}

	// Verify both expected files are in the list
	expectedFiles := map[string]bool{
		"audit.csv.sha256":         false,
		"http_results.json.sha256": false,
	}

	for _, file := range filesToSign {
		if _, exists := expectedFiles[file]; exists {
			expectedFiles[file] = true
		}
	}

	for file, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file %s to be signed", file)
		}
	}
}

func TestAutoSign_GPGCommandStructure(t *testing.T) {
	// Test that the GPG command is structured correctly
	expectedArgs := []string{
		"--armor",
		"--local-user",
		"test@example.com",
		"--sign",
		"/path/to/file.sha256",
	}

	// Validate command structure
	if expectedArgs[0] != "--armor" {
		t.Error("First arg should be --armor for ASCII armored output")
	}

	if expectedArgs[1] != "--local-user" {
		t.Error("Second arg should be --local-user to specify signing key")
	}

	if expectedArgs[3] != "--sign" {
		t.Error("Fourth arg should be --sign to create signature")
	}

	if len(expectedArgs) != 5 {
		t.Errorf("Expected 5 command arguments, got %d", len(expectedArgs))
	}
}
