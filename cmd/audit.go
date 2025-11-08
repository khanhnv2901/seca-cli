package cmd

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
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
func AppendAuditRow(engagementID string, operatorName string, commandName string, target string, status string, httpStatus int, tlsExpiry string, notes string, errMsg string, durationSeconds float64) error {
	// ensure engagement-specific directory under resultsDir
	dir := filepath.Join(resultsDir, engagementID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create results subdir failed: %w", err)
	}

	auditPath := filepath.Join(dir, "audit.csv")
	// check if file exists
	exists := true
	if _, err := os.Stat(auditPath); os.IsNotExist(err) {
		exists = false
	}

	f, err := os.OpenFile(auditPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open audit file failed: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	// if new file, write header first
	if !exists {
		_ = writer.Write(auditHeader)
		writer.Flush()
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

	_ = writer.Write(row)
	writer.Flush()

	return writer.Error()
}

// SaveRawCapture writes a limited raw HTTP response for auditing (be careful with PII)
func SaveRawCapture(engamentID, target string, headers map[string][]string, bodySnippet string) error {
	dir := filepath.Join(resultsDir, engamentID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
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
	fmt.Fprintf(f, "\n--- Body Snippet (max 2048 bytes) ---\n%s\n", bodySnippet)
	return nil
}

// HashFileSHA256 computes and writes a .sha256 companion file
func HashFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()

	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	hashPath := path + ".sha256"
	content := fmt.Sprintf("%s  %s\n", sum, filepath.Base(path))
	if err := os.WriteFile(hashPath, []byte(content), 0o644); err != nil {
		return "", err
	}
	return sum, nil
}
