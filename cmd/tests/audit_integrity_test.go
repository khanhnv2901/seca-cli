package tests

import (
	"bufio"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	cmdpkg "github.com/khanhnv2901/seca-cli/cmd"
	"github.com/khanhnv2901/seca-cli/cmd/testutil"
)

func TestAuditFileIntegrity(t *testing.T) {
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	for i := 0; i < 3; i++ {
		if err := cmdpkg.AppendAuditRow(
			env.AppCtx.ResultsDir,
			env.EngagementID,
			env.Operator,
			"check http",
			"https://example.com",
			"ok",
			200,
			time.Now().Add(30*time.Hour).UTC().Format(time.RFC3339),
			"",
			"",
			0.5,
		); err != nil {
			t.Fatalf("append audit row failed: %v", err)
		}
	}

	auditPath := filepath.Join(env.AppCtx.ResultsDir, env.EngagementID, "audit.csv")
	f, err := os.Open(auditPath)
	if err != nil {
		t.Fatalf("failed to open audit file: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(bufio.NewReader(f))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to read csv: %v", err)
	}

	if len(rows) != 4 {
		t.Fatalf("expected header + 3 rows, got %d rows", len(rows))
	}

	hash, err := cmdpkg.HashFileSHA256(auditPath)
	if err != nil {
		t.Fatalf("hashing audit file failed: %v", err)
	}

	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit file: %v", err)
	}
	expectedHash := sha256.Sum256(data)
	if hash != hex.EncodeToString(expectedHash[:]) {
		t.Fatalf("hash mismatch: expected %s, got %s", hex.EncodeToString(expectedHash[:]), hash)
	}

	hashFile := auditPath + ".sha256"
	if _, err := os.Stat(hashFile); err != nil {
		t.Fatalf("expected hash file to exist: %v", err)
	}
}
