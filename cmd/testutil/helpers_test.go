package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTestEnv(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Verify basic setup
	if env.TmpDir == "" {
		t.Error("TmpDir should not be empty")
	}

	if env.EngagementID == "" {
		t.Error("EngagementID should not be empty")
	}

	if env.Operator != "test-operator" {
		t.Errorf("Expected operator 'test-operator', got %s", env.Operator)
	}

	if env.AppCtx == nil {
		t.Fatal("AppCtx should not be nil")
	}

	// Verify directory was created
	if _, err := os.Stat(env.ResultsPath()); os.IsNotExist(err) {
		t.Error("Results directory should exist")
	}
}

func TestTestEnv_WithEngagementID(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	customID := "custom-engagement-123"
	env.WithEngagementID(customID)

	if env.EngagementID != customID {
		t.Errorf("Expected engagement ID %s, got %s", customID, env.EngagementID)
	}

	// Verify directory was created
	expectedPath := filepath.Join(env.AppCtx.ResultsDir, customID)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Directory %s should exist", expectedPath)
	}
}

func TestTestEnv_WithOperator(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	customOperator := "alice@example.com"
	env.WithOperator(customOperator)

	if env.Operator != customOperator {
		t.Errorf("Expected operator %s, got %s", customOperator, env.Operator)
	}

	if env.AppCtx.Operator != customOperator {
		t.Errorf("Expected AppCtx operator %s, got %s", customOperator, env.AppCtx.Operator)
	}
}

func TestTestEnv_CreateFile(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	content := []byte("test content")
	relativePath := "subdir/test.txt"

	filePath := env.CreateFile(relativePath, content)

	// Verify file exists
	if !env.FileExists(relativePath) {
		t.Error("File should exist")
	}

	// Verify content
	readContent := env.ReadFile(relativePath)
	if string(readContent) != string(content) {
		t.Errorf("Expected content %s, got %s", content, readContent)
	}

	// Verify full path
	expectedPath := filepath.Join(env.TmpDir, relativePath)
	if filePath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, filePath)
	}
}

func TestTestEnv_FileExists(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Test non-existent file
	if env.FileExists("nonexistent.txt") {
		t.Error("Non-existent file should return false")
	}

	// Create file and test again
	env.CreateFile("exists.txt", []byte("content"))
	if !env.FileExists("exists.txt") {
		t.Error("Existing file should return true")
	}
}

func TestTestEnv_MustExist(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create file
	env.CreateFile("test.txt", []byte("content"))

	// This should not fail
	env.MustExist("test.txt")

	// Test would fail with t.Fatalf for non-existent file, but we can't test that directly
}

func TestTestEnv_MustNotExist(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// This should not fail for non-existent file
	env.MustNotExist("nonexistent.txt")

	// Test would fail with t.Fatalf for existing file, but we can't test that directly
}

func TestTestEnv_AddCleanup(t *testing.T) {
	env := NewTestEnv(t)

	cleanupCalled := false
	env.AddCleanup(func() {
		cleanupCalled = true
	})

	// Call cleanup
	env.Cleanup()

	if !cleanupCalled {
		t.Error("Cleanup function should have been called")
	}
}

func TestTestEnv_AddCleanup_LIFO(t *testing.T) {
	env := NewTestEnv(t)

	order := []int{}

	env.AddCleanup(func() {
		order = append(order, 1)
	})
	env.AddCleanup(func() {
		order = append(order, 2)
	})
	env.AddCleanup(func() {
		order = append(order, 3)
	})

	env.Cleanup()

	// Cleanup should be LIFO: last added, first executed
	expectedOrder := []int{3, 2, 1}
	if len(order) != len(expectedOrder) {
		t.Fatalf("Expected %d cleanup calls, got %d", len(expectedOrder), len(order))
	}

	for i, expected := range expectedOrder {
		if order[i] != expected {
			t.Errorf("Cleanup order[%d]: expected %d, got %d", i, expected, order[i])
		}
	}
}

func TestTestEnv_ResultsPath(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	resultsPath := env.ResultsPath()
	expectedPath := filepath.Join(env.AppCtx.ResultsDir, env.EngagementID)

	if resultsPath != expectedPath {
		t.Errorf("Expected results path %s, got %s", expectedPath, resultsPath)
	}
}

func TestTestEnv_Chaining(t *testing.T) {
	env := NewTestEnv(t).
		WithEngagementID("chained-123").
		WithOperator("chained-operator")

	defer env.Cleanup()

	if env.EngagementID != "chained-123" {
		t.Errorf("Chained engagement ID not set correctly")
	}

	if env.Operator != "chained-operator" {
		t.Errorf("Chained operator not set correctly")
	}
}
