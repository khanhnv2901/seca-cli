package testutil

import (
	"os"
	"path/filepath"
	"testing"

	consts "github.com/khanhnv2901/seca-cli/internal/constants"
	"github.com/khanhnv2901/seca-cli/internal/security"
	"go.uber.org/zap"
)

// AppContext represents the application context (imported from parent package).
// This is redeclared here to avoid circular imports.
type AppContext struct {
	Logger     *zap.SugaredLogger
	Operator   string
	ResultsDir string
}

// TestEnv holds test environment configuration and cleanup functions.
type TestEnv struct {
	TmpDir       string
	EngagementID string
	Operator     string
	AppCtx       *AppContext
	cleanupFuncs []func()
	t            *testing.T
}

// NewTestEnv creates a new test environment with automatic cleanup.
// Usage:
//
//	env := testutil.NewTestEnv(t)
//	defer env.Cleanup()
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	tmpDir := t.TempDir() // Automatically cleaned up by Go test framework
	engagementID := "test-engagement-" + t.Name()

	env := &TestEnv{
		TmpDir:       tmpDir,
		EngagementID: engagementID,
		Operator:     "test-operator",
		t:            t,
		cleanupFuncs: []func(){},
	}

	// Create results directory structure
	resultsDir := filepath.Join(tmpDir, "results")
	if err := os.MkdirAll(filepath.Join(resultsDir, engagementID), consts.DefaultDirPerm); err != nil {
		t.Fatalf("Failed to create test results directory: %v", err)
	}

	// Initialize AppContext
	env.AppCtx = &AppContext{
		Logger:     nil, // Most tests don't need a real logger
		Operator:   env.Operator,
		ResultsDir: resultsDir,
	}

	return env
}

// WithEngagementID sets a custom engagement ID.
func (e *TestEnv) WithEngagementID(id string) *TestEnv {
	e.t.Helper()
	e.EngagementID = id

	// Create directory for new engagement
	if err := os.MkdirAll(filepath.Join(e.AppCtx.ResultsDir, id), consts.DefaultDirPerm); err != nil {
		e.t.Fatalf("Failed to create engagement directory: %v", err)
	}

	return e
}

// WithOperator sets a custom operator name.
func (e *TestEnv) WithOperator(operator string) *TestEnv {
	e.Operator = operator
	e.AppCtx.Operator = operator
	return e
}

// WithLogger sets a custom logger for tests that need one.
func (e *TestEnv) WithLogger(logger *zap.SugaredLogger) *TestEnv {
	e.AppCtx.Logger = logger
	return e
}

// AddCleanup adds a cleanup function to be called when Cleanup() is called.
// Cleanup functions are called in reverse order (LIFO).
func (e *TestEnv) AddCleanup(fn func()) {
	e.cleanupFuncs = append([]func(){fn}, e.cleanupFuncs...)
}

// Cleanup runs all registered cleanup functions.
// Typically called with defer: defer env.Cleanup()
func (e *TestEnv) Cleanup() {
	for _, fn := range e.cleanupFuncs {
		fn()
	}
}

// ResultsPath returns the full path to the results directory for the test engagement.
func (e *TestEnv) ResultsPath() string {
	return filepath.Join(e.AppCtx.ResultsDir, e.EngagementID)
}

// CreateFile creates a file in the test environment with the given content.
// The file path is relative to the test's temporary directory.
func (e *TestEnv) CreateFile(relativePath string, content []byte) string {
	e.t.Helper()

	fullPath := resolveTmpPath(e.TmpDir, relativePath, e.t)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, consts.DefaultDirPerm); err != nil {
		e.t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, content, consts.DefaultFilePerm); err != nil {
		e.t.Fatalf("Failed to create file %s: %v", fullPath, err)
	}

	return fullPath
}

// ReadFile reads a file from the test environment.
// The file path is relative to the test's temporary directory.
func (e *TestEnv) ReadFile(relativePath string) []byte {
	e.t.Helper()

	fullPath := resolveTmpPath(e.TmpDir, relativePath, e.t)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		e.t.Fatalf("Failed to read file %s: %v", fullPath, err)
	}

	return content
}

// FileExists checks if a file exists in the test environment.
func (e *TestEnv) FileExists(relativePath string) bool {
	fullPath := resolveTmpPath(e.TmpDir, relativePath, e.t)
	_, err := os.Stat(fullPath)
	return err == nil
}

// MustNotExist fails the test if the file exists.
func (e *TestEnv) MustNotExist(relativePath string) {
	e.t.Helper()
	if e.FileExists(relativePath) {
		e.t.Fatalf("File %s should not exist but does", relativePath)
	}
}

// MustExist fails the test if the file does not exist.
func (e *TestEnv) MustExist(relativePath string) {
	e.t.Helper()
	if !e.FileExists(relativePath) {
		e.t.Fatalf("File %s should exist but does not", relativePath)
	}
}

func resolveTmpPath(baseDir, relativePath string, t *testing.T) string {
	t.Helper()
	path, err := security.ResolveWithin(baseDir, relativePath)
	if err != nil {
		t.Fatalf("invalid test path %s: %v", relativePath, err)
	}
	return path
}

// SetupEngagementsFile creates a test engagements file with the given content.
// Returns a cleanup function that should be deferred.
func SetupEngagementsFile(t *testing.T, getFilePath func() (string, error)) func() {
	t.Helper()

	filePath, err := getFilePath()
	if err != nil {
		t.Fatalf("Failed to get engagements file path: %v", err)
	}

	// Backup existing file if it exists
	backupFile := filePath + ".test.backup"
	if _, err := os.Stat(filePath); err == nil {
		data, _ := os.ReadFile(filePath) // #nosec G304 -- engagements path resolved via getEngagementsFilePath within the controlled test dir.
		_ = os.WriteFile(backupFile, data, consts.DefaultFilePerm)
	}

	// Remove existing file
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove %s: %v", filePath, err)
	}

	// Return cleanup function
	return func() {
		// Restore backup if it existed
		if _, err := os.Stat(backupFile); err == nil {
			data, _ := os.ReadFile(backupFile) // #nosec G304 -- backup file created in the same directory as the trusted engagements file.
			_ = os.WriteFile(filePath, data, consts.DefaultFilePerm)
			if err := os.Remove(backupFile); err != nil && !os.IsNotExist(err) {
				t.Fatalf("failed to remove %s: %v", backupFile, err)
			}
		} else {
			// Just remove test file
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				t.Fatalf("failed to clean up %s: %v", filePath, err)
			}
		}
	}
}

// SetupAppContext sets up a test AppContext and returns a cleanup function.
// This is useful for tests that need to mock the global application context.
func SetupAppContext(t *testing.T, setGlobal func(*AppContext), getGlobal func() *AppContext) func() {
	t.Helper()

	originalAppCtx := getGlobal()
	testAppCtx := &AppContext{
		Logger:     nil,
		Operator:   "test-operator",
		ResultsDir: "/tmp/test-results",
	}
	setGlobal(testAppCtx)

	return func() {
		setGlobal(originalAppCtx)
	}
}
