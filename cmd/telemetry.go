package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/infrastructure/checker"
	consts "github.com/khanhnv2901/seca-cli/internal/shared/constants"
)

type TelemetryRecord struct {
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

	record := TelemetryRecord{
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

	if _, err := ensureResultsDir(appCtx.ResultsDir, engagementID); err != nil {
		return fmt.Errorf("prepare telemetry directory: %w", err)
	}

	telemetryPath, err := resolveResultsPath(appCtx.ResultsDir, engagementID, "telemetry.jsonl")
	if err != nil {
		return fmt.Errorf("determine telemetry path: %w", err)
	}
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

func loadTelemetryHistory(resultsDir, engagementID string, limit int) ([]TelemetryRecord, error) {
	if limit <= 0 {
		limit = 5
	}

	telemetryPath, err := resolveResultsPath(resultsDir, engagementID, "telemetry.jsonl")
	if err != nil {
		return nil, fmt.Errorf("invalid telemetry path: %w", err)
	}
	f, err := os.Open(telemetryPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	records := make([]TelemetryRecord, 0, limit)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec TelemetryRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		if rec.EngagementID != engagementID {
			continue
		}
		records = append(records, rec)
	}

	// Keep only most recent limit entries
	if len(records) > limit {
		records = records[len(records)-limit:]
	}

	return records, scanner.Err()
}
