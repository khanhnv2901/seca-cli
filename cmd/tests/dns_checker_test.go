package tests

import (
	"context"
	"testing"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/checker"
)

func TestDNSCheckerResolution(t *testing.T) {
	t.Parallel()

	dnsChecker := &checker.DNSChecker{
		Timeout: 2 * time.Second,
	}

	result := dnsChecker.Check(context.Background(), "localhost")
	if result.Status != "ok" {
		t.Fatalf("expected status ok for localhost, got %s (error: %s)", result.Status, result.Error)
	}

	aRecords, ok := result.DNSRecords["a_records"].([]string)
	if !ok {
		t.Fatalf("expected a_records to be []string, got %T", result.DNSRecords["a_records"])
	}
	if len(aRecords) == 0 {
		t.Fatalf("expected at least one A record, got 0")
	}
}
