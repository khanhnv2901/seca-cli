package cmd

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/khanhnv2901/seca-cli/cmd/testutil"
	"github.com/khanhnv2901/seca-cli/internal/checker"
)

func TestRecordTelemetry_WritesMetrics(t *testing.T) {
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	results := []checker.CheckResult{
		{Status: "ok"},
		{Status: "error"},
		{Status: "ok"},
	}

	appCtx := &AppContext{
		Operator:   env.Operator,
		ResultsDir: env.AppCtx.ResultsDir,
	}

	if err := recordTelemetry(appCtx, "eng-123", "check http", results, 3*time.Second); err != nil {
		t.Fatalf("recordTelemetry returned error: %v", err)
	}

	path := filepath.Join(env.AppCtx.ResultsDir, "telemetry.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open telemetry file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		t.Fatalf("expected telemetry record, file empty")
	}

	var rec telemetryRecord
	if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
		t.Fatalf("failed to unmarshal record: %v", err)
	}

	if rec.EngagementID != "eng-123" {
		t.Errorf("expected engagement_id eng-123, got %s", rec.EngagementID)
	}

	if rec.SuccessCount != 2 || rec.ErrorCount != 1 {
		t.Errorf("unexpected counts: %+v", rec)
	}

	expectedRate := (2.0 / 3.0) * 100
	if math.Abs(rec.SuccessRate-expectedRate) > 0.0001 {
		t.Errorf("expected success rate %.6f, got %.6f", expectedRate, rec.SuccessRate)
	}

	if rec.DurationSeconds != 3 {
		t.Errorf("expected duration 3s, got %f", rec.DurationSeconds)
	}
}
