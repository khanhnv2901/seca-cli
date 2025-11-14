package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestStoreAndGetAppContext(t *testing.T) {
	original := globalAppContext
	defer func() {
		globalAppContext = original
	}()

	cmd := &cobra.Command{Use: "root"}
	appCtx := &AppContext{Operator: "tester"}

	storeAppContext(cmd, appCtx)

	got := getAppContext(cmd)
	if got != appCtx {
		t.Fatalf("expected stored app context to be returned")
	}
}
