package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateEngagementID(t *testing.T) {
	valid := []string{"abc123", "ENG-001"}
	for _, id := range valid {
		if err := validateEngagementID(id); err != nil {
			t.Fatalf("expected id %s to be valid: %v", id, err)
		}
	}

	invalid := []string{"", ".", "..", "bad/id", `bad\id`}
	for _, id := range invalid {
		if err := validateEngagementID(id); err == nil {
			t.Fatalf("expected id %s to be rejected", id)
		}
	}
}

func TestResolveAndEnsureResultsPath(t *testing.T) {
	base := t.TempDir()
	path, err := resolveResultsPath(base, "eng123", "http_results.json")
	if err != nil {
		t.Fatalf("resolveResultsPath failed: %v", err)
	}
	if filepath.Dir(path) != filepath.Join(base, "eng123") {
		t.Fatalf("path resolved outside engagement dir: %s", path)
	}

	dir, err := ensureResultsDir(base, "eng123")
	if err != nil {
		t.Fatalf("ensureResultsDir failed: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected directory %s to exist: %v", dir, err)
	}
}

func TestValidateGPGKey(t *testing.T) {
	if err := validateGPGKey("user@example.com"); err != nil {
		t.Fatalf("unexpected error for valid key: %v", err)
	}
	if err := validateGPGKey(""); err == nil {
		t.Fatal("expected error for empty key")
	}
	if err := validateGPGKey("key\nbad"); err == nil {
		t.Fatal("expected error for key with newline")
	}
}

func TestResolveResultsPathInvalidID(t *testing.T) {
	if _, err := resolveResultsPath(t.TempDir(), "bad/id"); err == nil {
		t.Fatal("expected error for invalid engagement id")
	}
}

func TestEnsureResultsDirErrorsWhenBaseIsFile(t *testing.T) {
	base := filepath.Join(t.TempDir(), "results-file")
	if err := os.WriteFile(base, []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to create base file: %v", err)
	}
	if _, err := ensureResultsDir(base, "eng123"); err == nil {
		t.Fatal("expected ensureResultsDir to fail when base is not a directory")
	}
}
