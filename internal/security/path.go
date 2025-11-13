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
