package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/checker"
	consts "github.com/khanhnv2901/seca-cli/internal/constants"
)

type telemetryRecord struct {
	Timestamp           time.Time `json:"timestamp"`
	Command             string    `json:"command"`
	EngagementID        string    `json:"engagement_id"`
	TargetCount         int       `json:"target_count"`
	SuccessCount        int       `json:"success_count"`
	ErrorCount          int       `json:"error_count"`
	SuccessRate         float64   `json:"success_rate"`
	DurationSeconds     float64   `json:"duration_seconds"`
	AvgDurationPerCheck float64   `json:"avg_duration_per_check"`
}

func recordTelemetry(appCtx *AppContext, engagementID string, command string, results []checker.CheckResult, duration time.Duration) error {
	okCount, errorCount := summarizeStatuses(results)
	total := len(results)

	successRate := 0.0
	if total > 0 {
		successRate = (float64(okCount) / float64(total)) * 100
	}

	avgDuration := 0.0
	if total > 0 {
		avgDuration = duration.Seconds() / float64(total)
	}

	record := telemetryRecord{
		Timestamp:           time.Now().UTC(),
		Command:             command,
		EngagementID:        engagementID,
		TargetCount:         total,
		SuccessCount:        okCount,
		ErrorCount:          errorCount,
		SuccessRate:         successRate,
		DurationSeconds:     duration.Seconds(),
		AvgDurationPerCheck: avgDuration,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal telemetry: %w", err)
	}

	telemetryPath := filepath.Join(appCtx.ResultsDir, "telemetry.jsonl")
	f, err := os.OpenFile(telemetryPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, consts.DefaultFilePerm)
	if err != nil {
		return fmt.Errorf("open telemetry file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write telemetry: %w", err)
	}

	return nil
}

func summarizeStatuses(results []checker.CheckResult) (okCount, errorCount int) {
	for _, r := range results {
		if r.Status == "ok" {
			okCount++
		} else {
			errorCount++
		}
	}
	return okCount, errorCount
}
