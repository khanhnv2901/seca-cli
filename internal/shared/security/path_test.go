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

func TestResolveWithinEmptyBase(t *testing.T) {
	_, err := ResolveWithin("", "some", "path")
	if err == nil {
		t.Fatal("expected error for empty base directory")
	}
	if err.Error() != "base directory is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestResolveWithinMultipleElements(t *testing.T) {
	base := t.TempDir()

	resolved, err := ResolveWithin(base, "a", "b", "c", "file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(base, "a", "b", "c", "file.txt")
	if resolved != expected {
		t.Errorf("expected %s, got %s", expected, resolved)
	}

	// Verify it's within base
	if !strings.HasPrefix(resolved, base) {
		t.Error("resolved path should be within base")
	}
}

func TestResolveWithinAbsolutePathAttempt(t *testing.T) {
	base := t.TempDir()

	// Try to use absolute path in elements (should still be joined safely)
	resolved, err := ResolveWithin(base, "/etc/passwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should resolve within base, not to /etc/passwd
	if !strings.HasPrefix(resolved, base) {
		t.Errorf("resolved path %s should be within base %s", resolved, base)
	}
}

func TestResolveWithinDotDotInMiddle(t *testing.T) {
	base := t.TempDir()

	// This is safe: a/b/../c resolves to a/c within base
	resolved, err := ResolveWithin(base, "a", "b", "..", "c")
	if err != nil {
		t.Fatalf("unexpected error for safe traversal: %v", err)
	}

	expected := filepath.Join(base, "a", "c")
	if resolved != expected {
		t.Errorf("expected %s, got %s", expected, resolved)
	}
}

func TestResolveWithinSymbolicEscape(t *testing.T) {
	base := t.TempDir()

	// Multiple .. attempts
	tests := []struct {
		name  string
		elems []string
	}{
		{"double escape", []string{"..", ".."}},
		{"triple escape", []string{"..", "..", ".."}},
		{"escape with path", []string{"..", "..", "etc", "passwd"}},
		{"relative escape", []string{"a", "..", "..", "etc"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveWithin(base, tt.elems...)
			if err == nil {
				t.Error("expected path escape error")
			}
			if !strings.Contains(err.Error(), "escapes base directory") {
				t.Errorf("expected escape error, got: %v", err)
			}
		})
	}
}

func TestResolveWithinSingleDot(t *testing.T) {
	base := t.TempDir()

	// Single dot should resolve to base itself
	resolved, err := ResolveWithin(base, ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved != base {
		t.Errorf("expected %s, got %s", base, resolved)
	}
}

func TestResolveWithinNoElements(t *testing.T) {
	base := t.TempDir()

	// No elements should resolve to base
	resolved, err := ResolveWithin(base)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved != base {
		t.Errorf("expected %s, got %s", base, resolved)
	}
}

func TestResolveWithinComplexPath(t *testing.T) {
	base := t.TempDir()

	// Complex but safe path
	resolved, err := ResolveWithin(base, "./a/./b/../c/./d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should clean to base/a/c/d
	expected := filepath.Join(base, "a", "c", "d")
	if resolved != expected {
		t.Errorf("expected %s, got %s", expected, resolved)
	}
}

func TestResolveWithinErrorHandling(t *testing.T) {
	base := t.TempDir()

	// Test various edge cases
	tests := []struct {
		name    string
		base    string
		elems   []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty base",
			base:    "",
			elems:   []string{"file.txt"},
			wantErr: true,
			errMsg:  "base directory is required",
		},
		{
			name:    "escape attempt",
			base:    base,
			elems:   []string{"..", "outside"},
			wantErr: true,
			errMsg:  "escapes base directory",
		},
		{
			name:    "valid path",
			base:    base,
			elems:   []string{"valid", "file.txt"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveWithin(tt.base, tt.elems...)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
