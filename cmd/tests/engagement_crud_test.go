package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/khanhnv2901/seca-cli/internal/application"
)

func TestEngagementCRUD(t *testing.T) {
	dataDir := t.TempDir()
	resultsDir := filepath.Join(dataDir, "results")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatalf("failed to create results directory: %v", err)
	}

	container, err := application.NewContainer(dataDir, resultsDir)
	if err != nil {
		t.Fatalf("failed to initialize services: %v", err)
	}

	ctx := context.Background()
	service := container.EngagementService

	eng, err := service.CreateEngagement(ctx, "CRUD Test", "owner@example.com", "Test ROE", nil)
	if err != nil {
		t.Fatalf("create engagement failed: %v", err)
	}

	if err := service.AcknowledgeROE(ctx, eng.ID()); err != nil {
		t.Fatalf("acknowledge ROE failed: %v", err)
	}

	if err := service.AddToScope(ctx, eng.ID(), []string{"https://example.com"}); err != nil {
		t.Fatalf("add scope failed: %v", err)
	}

	updated, err := service.GetEngagement(ctx, eng.ID())
	if err != nil {
		t.Fatalf("get engagement failed: %v", err)
	}
	if len(updated.Scope()) != 1 || updated.Scope()[0] != "https://example.com" {
		t.Fatalf("expected scope entry to be added, got %+v", updated.Scope())
	}

	if err := service.DeleteEngagement(ctx, eng.ID()); err != nil {
		t.Fatalf("delete engagement failed: %v", err)
	}

	list, err := service.ListEngagements(ctx)
	if err != nil {
		t.Fatalf("list engagements failed: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no engagements after delete, got %d", len(list))
	}
}
