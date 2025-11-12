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
		Config:     newCLIConfig(),
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

	var rec TelemetryRecord
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

func TestLoadTelemetryHistory(t *testing.T) {
	env := testutil.NewTestEnv(t)
	defer env.Cleanup()

	path := filepath.Join(env.AppCtx.ResultsDir, "telemetry.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create telemetry file: %v", err)
	}
	encoder := json.NewEncoder(f)
	records := []TelemetryRecord{
		{EngagementID: "eng-1", SuccessRate: 50, DurationSeconds: 2, Timestamp: time.Now().Add(-3 * time.Hour), Command: "check http"},
		{EngagementID: "eng-2", SuccessRate: 30, DurationSeconds: 4, Timestamp: time.Now().Add(-2 * time.Hour), Command: "check dns"},
		{EngagementID: "eng-1", SuccessRate: 70, DurationSeconds: 3, Timestamp: time.Now().Add(-1 * time.Hour), Command: "check http"},
		{EngagementID: "eng-1", SuccessRate: 90, DurationSeconds: 2.5, Timestamp: time.Now(), Command: "check http"},
	}
	for _, rec := range records {
		if err := encoder.Encode(rec); err != nil {
			t.Fatalf("failed to encode telemetry: %v", err)
		}
	}
	f.Close()

	history, err := loadTelemetryHistory(env.AppCtx.ResultsDir, "eng-1", 2)
	if err != nil {
		t.Fatalf("loadTelemetryHistory returned error: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}

	if history[0].SuccessRate != 70 || history[1].SuccessRate != 90 {
		t.Fatalf("unexpected history order: %v", history)
	}

	for _, rec := range history {
		if rec.EngagementID != "eng-1" {
			t.Fatalf("unexpected engagement id in history: %s", rec.EngagementID)
		}
	}
}
