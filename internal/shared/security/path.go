package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrPathEscape indicates the resolved path would escape the trusted root directory.
	ErrPathEscape = errors.New("path escapes base directory")
)

// ResolveWithin joins the provided path elements under the given base directory and ensures
// the resulting path never traverses outside of that base. The returned path is absolute.
func ResolveWithin(base string, elems ...string) (string, error) {
	if base == "" {
		return "", errors.New("base directory is required")
	}

	cleanBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve base path: %w", err)
	}

	joined := filepath.Join(append([]string{cleanBase}, elems...)...)
	target, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("resolve target path: %w", err)
	}

	rel, err := filepath.Rel(cleanBase, target)
	if err != nil {
		return "", fmt.Errorf("relativize path: %w", err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %s", ErrPathEscape, target)
	}

	return target, nil
}

// IsValidPath checks if a path is valid and does not contain path traversal attempts
func IsValidPath(path string) bool {
	if path == "" {
		return false
	}

	// Check for path traversal patterns
	if strings.Contains(path, "..") {
		return false
	}

	// Ensure the path is absolute or can be made absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Basic sanity check - path should not be empty after cleaning
	cleanPath := filepath.Clean(absPath)
	return cleanPath != "" && cleanPath != "/"
}
