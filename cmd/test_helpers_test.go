package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/khanhnv2901/seca-cli/internal/application"
	consts "github.com/khanhnv2901/seca-cli/internal/shared/constants"
)

// setupTestAppContext initializes a minimal AppContext for tests that don't need services.
func setupTestAppContext(t *testing.T) func() {
	t.Helper()
	return setupTestAppContextWithOptions(t, false)
}

// setupTestAppContextWithServices initializes an AppContext with real DDD services.
func setupTestAppContextWithServices(t *testing.T) func() {
	t.Helper()
	return setupTestAppContextWithOptions(t, true)
}

func setupTestAppContextWithOptions(t *testing.T, includeServices bool) func() {
	t.Helper()

	original := globalAppContext

	dataDir := os.Getenv(dataDirEnvVar)
	if dataDir == "" {
		dataDir = t.TempDir()
		t.Setenv(dataDirEnvVar, dataDir)
	} else {
		// Ensure the directory exists if the test already set the env var.
		if err := os.MkdirAll(dataDir, consts.DefaultDirPerm); err != nil {
			t.Fatalf("failed to create data directory: %v", err)
		}
	}

	resultsDir := filepath.Join(dataDir, "results")
	if err := os.MkdirAll(resultsDir, consts.DefaultDirPerm); err != nil {
		t.Fatalf("failed to create results directory: %v", err)
	}

	appCtx := &AppContext{
		Logger:     nil,
		Operator:   "test-operator",
		ResultsDir: resultsDir,
		Config:     newCLIConfig(),
	}

	if includeServices {
		services, err := application.NewContainer(dataDir, resultsDir)
		if err != nil {
			t.Fatalf("failed to initialize services: %v", err)
		}
		appCtx.Services = services
	}

	globalAppContext = appCtx

	return func() {
		globalAppContext = original
	}
}
