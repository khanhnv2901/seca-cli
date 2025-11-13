package cmd

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestApplyIntDefault(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Int("timeout", 0, "")

	var applied int
	applyIntDefault(flags, "timeout", 15, func(v int) {
		applied = v
	})
	if applied != 15 {
		t.Fatalf("expected setter to receive 15, got %d", applied)
	}

	// When flag already set, setter should not run.
	if err := flags.Set("timeout", "7"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}
	applied = 0
	applyIntDefault(flags, "timeout", 20, func(v int) {
		applied = v
	})
	if applied != 0 {
		t.Fatalf("setter should not run when flag overridden, got %d", applied)
	}
}

func TestApplyBoolDefault(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Bool("telemetry", false, "")

	applied := false
	applyBoolDefault(flags, "telemetry", true, func(v bool) {
		applied = v
	})
	if !applied {
		t.Fatal("expected setter to run with true")
	}

	if err := flags.Set("telemetry", "false"); err != nil {
		t.Fatalf("failed to set bool flag: %v", err)
	}
	applied = true
	applyBoolDefault(flags, "telemetry", true, func(v bool) {
		applied = v
	})
	if !applied {
		t.Fatalf("setter should not change value when flag already set")
	}
}

func TestSetStringFlagIfUnset(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("operator", "", "")

	setStringFlagIfUnset(flags, "operator", "default-operator")
	if got := flags.Lookup("operator").Value.String(); got != "default-operator" {
		t.Fatalf("expected operator to be default, got %s", got)
	}

	if err := flags.Set("operator", "user-provided"); err != nil {
		t.Fatalf("failed to set operator: %v", err)
	}
	setStringFlagIfUnset(flags, "operator", "new-default")
	if got := flags.Lookup("operator").Value.String(); got != "user-provided" {
		t.Fatalf("expected operator to remain user-provided, got %s", got)
	}
}

func TestDetectOperatorFromEnv(t *testing.T) {
	t.Setenv("USER", "env-user")
	if got := detectOperatorFromEnv(); got != "env-user" {
		t.Fatalf("expected env-user, got %s", got)
	}

	t.Setenv("USER", "")
	t.Setenv("LOGNAME", "log-user")
	if got := detectOperatorFromEnv(); got != "log-user" {
		t.Fatalf("expected log-user, got %s", got)
	}
}

func TestNewCLIConfigDefaults(t *testing.T) {
	cfg := newCLIConfig()
	if cfg.Check.TimeoutSecs != defaultHTTPTimeoutSeconds {
		t.Fatalf("unexpected timeout default: %d", cfg.Check.TimeoutSecs)
	}
	if cfg.Check.DNS.Timeout != defaultDNSTimeoutSeconds {
		t.Fatalf("unexpected DNS timeout: %d", cfg.Check.DNS.Timeout)
	}
	if cfg.Check.Crawl.MaxDepth != 2 {
		t.Fatalf("unexpected crawl depth: %d", cfg.Check.Crawl.MaxDepth)
	}
	if cfg.Check.Crawl.MaxPages != 50 {
		t.Fatalf("unexpected crawl max pages: %d", cfg.Check.Crawl.MaxPages)
	}
	if cfg.Check.Crawl.JSWaitTime != 2 {
		t.Fatalf("unexpected crawl JS wait time: %d", cfg.Check.Crawl.JSWaitTime)
	}
	if !cfg.Check.Crawl.AutoDetectJS {
		t.Fatalf("expected auto-detect JS to be enabled by default")
	}
}
