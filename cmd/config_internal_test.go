package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
	if cfg.Check.Network.EnablePortScan {
		t.Fatalf("expected network port scanning to be disabled by default")
	}
	if cfg.Check.Network.PortScanTimeout != defaultPortScanTimeoutSecs {
		t.Fatalf("unexpected default port scan timeout: %d", cfg.Check.Network.PortScanTimeout)
	}
	if cfg.Check.Network.MaxPortWorkers != defaultPortScanWorkers {
		t.Fatalf("unexpected default port worker count: %d", cfg.Check.Network.MaxPortWorkers)
	}
}

func TestLoadDefaultOverrides(t *testing.T) {
	t.Cleanup(viper.Reset)

	viper.Set("defaults.timeout_secs", 30)
	viper.Set("defaults.telemetry", true)
	viper.Set("defaults.operator", "config-operator")
	viper.Set("defaults.retention_days", 45)
	viper.Set("defaults.hash_algorithm", "sha512")
	viper.Set("defaults.secure_results", true)

	overrides := loadDefaultOverrides()

	if overrides.TimeoutSecs == nil || *overrides.TimeoutSecs != 30 {
		t.Fatalf("expected timeout override 30, got %+v", overrides.TimeoutSecs)
	}
	if overrides.TelemetryEnabled == nil || !*overrides.TelemetryEnabled {
		t.Fatalf("expected telemetry override true, got %+v", overrides.TelemetryEnabled)
	}
	if overrides.Operator != "config-operator" || !overrides.OperatorOverride {
		t.Fatalf("expected operator override to be set, got %+v", overrides)
	}
	if overrides.RetentionDays == nil || *overrides.RetentionDays != 45 {
		t.Fatalf("expected retention override 45, got %+v", overrides.RetentionDays)
	}
	if overrides.HashAlgorithm != "sha512" {
		t.Fatalf("expected hash override sha512, got %s", overrides.HashAlgorithm)
	}
	if overrides.SecureResults == nil || !*overrides.SecureResults {
		t.Fatalf("expected secure results override true, got %+v", overrides.SecureResults)
	}
}

func TestApplyConfigDefaults(t *testing.T) {
	t.Cleanup(func() {
		viper.Reset()
		*cliConfig = *newCLIConfig()
	})

	*cliConfig = *newCLIConfig()

	viper.Set("defaults.timeout_secs", 20)
	viper.Set("defaults.telemetry", true)
	viper.Set("defaults.operator", "cfg-operator")
	viper.Set("defaults.retention_days", 90)
	viper.Set("defaults.hash_algorithm", "sha512")
	viper.Set("defaults.secure_results", true)

	// Reset flag state to simulate untouched CLI flags.
	if flag := checkCmd.PersistentFlags().Lookup("timeout"); flag != nil {
		flag.Changed = false
	}
	if flag := checkCmd.PersistentFlags().Lookup("telemetry"); flag != nil {
		flag.Changed = false
	}
	if flag := checkHTTPCmd.Flags().Lookup("retention-days"); flag != nil {
		flag.Changed = false
	}

	testCmd := &cobra.Command{Use: "root"}
	testCmd.Flags().String("operator", "", "")

	applyConfigDefaults(testCmd)

	if cliConfig.Defaults.TimeoutSecs != 20 || cliConfig.Check.TimeoutSecs != 20 {
		t.Fatalf("expected timeout defaults to update to 20, got %d/%d", cliConfig.Defaults.TimeoutSecs, cliConfig.Check.TimeoutSecs)
	}
	if !cliConfig.Defaults.TelemetryEnabled || !cliConfig.Check.TelemetryEnabled {
		t.Fatalf("expected telemetry defaults to be enabled")
	}
	if cliConfig.Defaults.RetentionDays != 90 || cliConfig.Check.RetentionDays != 90 {
		t.Fatalf("expected retention defaults to be 90, got %d/%d", cliConfig.Defaults.RetentionDays, cliConfig.Check.RetentionDays)
	}
	if cliConfig.Check.HashAlgorithm != "sha512" {
		t.Fatalf("expected hash algorithm sha512, got %s", cliConfig.Check.HashAlgorithm)
	}
	if !cliConfig.Check.SecureResults {
		t.Fatalf("expected secure results to be enabled")
	}

	if got := testCmd.Flags().Lookup("operator").Value.String(); got != "cfg-operator" {
		t.Fatalf("expected operator flag to be set by defaults, got %s", got)
	}
}
