package cmd

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	consts "github.com/khanhnv2901/seca-cli/internal/constants"
)

// audit header fields:
var auditHeader = []string{
	"timestamp",
	"engagement_id",
	"operator",
	"command",
	"target",
	"status",
	"http_status",
	"tls_expiry",
	"notes",
	"error",
	"duration_seconds",
}

// AppendAuditRow appends a single audit row to results/<engagementID>/audit.csv
func AppendAuditRow(resultsDir string, engagementID string, operatorName string, commandName string, target string, status string, httpStatus int, tlsExpiry string, notes string, errMsg string, durationSeconds float64) error {
	// ensure engagement-specific directory under resultsDir
	dir := filepath.Join(resultsDir, engagementID)
	if err := os.MkdirAll(dir, consts.DefaultDirPerm); err != nil {
		return fmt.Errorf("create results subdir failed: %w", err)
	}

	auditPath := filepath.Join(dir, "audit.csv")
	// check if file exists
	exists := true
	if _, err := os.Stat(auditPath); os.IsNotExist(err) {
		exists = false
	}

	f, err := os.OpenFile(auditPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, consts.DefaultFilePerm)
	if err != nil {
		return fmt.Errorf("open audit file failed: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// if new file, write header first
	if !exists {
		if err := writer.Write(auditHeader); err != nil {
			return fmt.Errorf("write audit header failed: %w", err)
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return fmt.Errorf("flush audit header failed: %w", err)
		}
	}

	row := []string{
		time.Now().UTC().Format(time.RFC3339),
		engagementID,
		operatorName,
		commandName,
		target,
		status,
		fmt.Sprintf("%d", httpStatus),
		tlsExpiry,
		notes,
		errMsg,
		fmt.Sprintf("%.3f", durationSeconds),
	}

	if err := writer.Write(row); err != nil {
		return fmt.Errorf("write audit row failed: %w", err)
	}
	writer.Flush()

	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush audit data failed: %w", err)
	}

	return nil
}

// SaveRawCapture writes a limited raw HTTP response for auditing (be careful with PII)
func SaveRawCapture(resultsDir string, engamentID, target string, headers map[string][]string, bodySnippet string) error {
	dir := filepath.Join(resultsDir, engamentID)
	if err := os.MkdirAll(dir, consts.DefaultDirPerm); err != nil {
		return err
	}
	filename := fmt.Sprintf("raw_%d.txt", time.Now().UnixNano())
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "Target: %s\nCaptureAt: %s\n\nHeaders:\n", target, time.Now().UTC().Format(time.RFC3339))
	for k, v := range headers {
		fmt.Fprintf(f, "%s: %s\n", k, v)
	}
	fmt.Fprintf(f, "\n--- Body Snippet (max %d bytes) ---\n%s\n", consts.RawCaptureLimitBytes, bodySnippet)
	return nil
}

// HashFileSHA256 computes and writes a .sha256 companion file
func HashFileSHA256(path string) (string, error) {
	return HashFile(path, HashAlgorithmSHA256)
}

// HashFile computes and writes a companion file for the given algorithm.
func HashFile(path string, algorithm HashAlgorithm) (string, error) {
	hasher, err := algorithm.newHasher()
	if err != nil {
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	hashPath := path + algorithm.FileExtension()
	content := fmt.Sprintf("%s  %s\n", sum, filepath.Base(path))
	if err := os.WriteFile(hashPath, []byte(content), consts.DefaultFilePerm); err != nil {
		return "", err
	}
	return sum, nil
}

// HashAlgorithm represents supported hashing algorithms for integrity files.
type HashAlgorithm string

const (
	HashAlgorithmSHA256 HashAlgorithm = "sha256"
	HashAlgorithmSHA512 HashAlgorithm = "sha512"
)

// ParseHashAlgorithm normalizes and validates the requested algorithm.
func ParseHashAlgorithm(raw string) (HashAlgorithm, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sha256", "":
		return HashAlgorithmSHA256, nil
	case "sha512":
		return HashAlgorithmSHA512, nil
	default:
		return "", fmt.Errorf("unsupported hash algorithm %q (use sha256 or sha512)", raw)
	}
}

func (h HashAlgorithm) String() string {
	if h == "" {
		return string(HashAlgorithmSHA256)
	}
	return string(h)
}

func (h HashAlgorithm) DisplayName() string {
	return strings.ToUpper(h.String())
}

func (h HashAlgorithm) FileExtension() string {
	return "." + h.String()
}

func (h HashAlgorithm) SumCommand() string {
	return fmt.Sprintf("%ssum", h.String())
}

func (h HashAlgorithm) newHasher() (hash.Hash, error) {
	switch h {
	case HashAlgorithmSHA256, "":
		return sha256.New(), nil
	case HashAlgorithmSHA512:
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm %q", h)
	}
}
