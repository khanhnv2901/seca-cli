package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWithinValidPath(t *testing.T) {
	base := t.TempDir()

	child := filepath.Join("sub", "file.txt")
	resolved, err := ResolveWithin(base, child)
	if err != nil {
		t.Fatalf("ResolveWithin returned error: %v", err)
	}
	if !strings.HasPrefix(resolved, base) {
		t.Fatalf("expected resolved path %s to stay within base %s", resolved, base)
	}

	// ensure path is actually usable
	if err := os.MkdirAll(filepath.Dir(resolved), 0o700); err != nil {
		t.Fatalf("failed to create parent dirs: %v", err)
	}
	if err := os.WriteFile(resolved, []byte("ok"), 0o600); err != nil {
		t.Fatalf("failed to write resolved file: %v", err)
	}
}

func TestResolveWithinBlocksEscape(t *testing.T) {
	base := t.TempDir()
	if _, err := ResolveWithin(base, "..", "etc", "passwd"); err == nil {
		t.Fatal("expected path escape error")
	}
}
