package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	consts "github.com/khanhnv2901/seca-cli/internal/shared/constants"
	"github.com/khanhnv2901/seca-cli/internal/shared/security"
)

// validateEngagementID ensures engagement identifiers can't be used for path traversal
// or command injection. IDs are stored inside filenames, so reject separators.
func validateEngagementID(id string) error {
	switch id {
	case "":
		return errors.New("engagement ID is required")
	case ".", "..":
		return fmt.Errorf("engagement ID %q is reserved", id)
	}
	if strings.ContainsAny(id, "/\\") {
		return fmt.Errorf("engagement ID %q must not contain path separators", id)
	}
	return nil
}

func resolveResultsPath(resultsDir, engagementID string, parts ...string) (string, error) {
	if err := validateEngagementID(engagementID); err != nil {
		return "", err
	}
	pathParts := append([]string{engagementID}, parts...)
	return security.ResolveWithin(resultsDir, pathParts...)
}

func ensureResultsDir(resultsDir, engagementID string) (string, error) {
	path, err := resolveResultsPath(resultsDir, engagementID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(path, consts.DefaultDirPerm); err != nil {
		return "", fmt.Errorf("create results directory: %w", err)
	}
	return path, nil
}

func validateGPGKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("gpg key is required")
	}
	if strings.ContainsAny(key, "\r\n") {
		return fmt.Errorf("gpg key %q contains invalid newline characters", key)
	}
	return nil
}
