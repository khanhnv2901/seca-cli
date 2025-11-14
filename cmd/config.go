package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultHTTPTimeoutSeconds  = 10
	defaultDNSTimeoutSeconds   = 10
	defaultPortScanTimeoutSecs = 2
	defaultPortScanWorkers     = 10
)

// CLIConfig captures runtime configuration shared across commands.
type CLIConfig struct {
	Defaults DefaultValues
	Check    CheckRuntimeConfig
}

// DefaultValues represent operator-level defaults, typically derived from env/config.
type DefaultValues struct {
	TimeoutSecs      int
	TelemetryEnabled bool
	Operator         string
	RetentionDays    int
}

// CheckRuntimeConfig consolidates flag-driven settings for check commands.
type CheckRuntimeConfig struct {
	Concurrency      int
	RateLimit        int
	TimeoutSecs      int
	AuditAppendRaw   bool
	RetentionDays    int
	AutoSign         bool
	GPGKey           string
	TelemetryEnabled bool
	ProgressEnabled  bool
	HashAlgorithm    string
	SecureResults    bool
	RetryCount       int
	DNS              DNSConfig
	Crawl            CrawlConfig
	Network          NetworkConfig
}

// DNSConfig groups DNS-specific runtime options.
type DNSConfig struct {
	Nameservers []string
	Timeout     int
}

// CrawlConfig captures HTTP crawl/discovery options.
type CrawlConfig struct {
	Enabled      bool
	MaxDepth     int
	MaxPages     int
	EnableJS     bool
	JSWaitTime   int // Time in seconds to wait for JavaScript to render
	AutoDetectJS bool
}

// NetworkConfig captures network checker runtime options.
type NetworkConfig struct {
	EnablePortScan  bool
	PortScanTimeout int
	Ports           []int
	MaxPortWorkers  int
}

type defaultOverrides struct {
	TimeoutSecs      *int
	TelemetryEnabled *bool
	Operator         string
	OperatorOverride bool
	RetentionDays    *int
	HashAlgorithm    string
	SecureResults    *bool
}

var cliConfig = newCLIConfig()

func newCLIConfig() *CLIConfig {
	operator := detectOperatorFromEnv()
	return &CLIConfig{
		Defaults: DefaultValues{
			TimeoutSecs:      defaultHTTPTimeoutSeconds,
			TelemetryEnabled: false,
			Operator:         operator,
			RetentionDays:    0,
		},
		Check: CheckRuntimeConfig{
			Concurrency:      1,
			RateLimit:        1,
			TimeoutSecs:      defaultHTTPTimeoutSeconds,
			TelemetryEnabled: false,
			RetentionDays:    0,
			HashAlgorithm:    HashAlgorithmSHA256.String(),
			DNS: DNSConfig{
				Nameservers: []string{},
				Timeout:     defaultDNSTimeoutSeconds,
			},
			Crawl: CrawlConfig{
				Enabled:      false,
				MaxDepth:     2,
				MaxPages:     50,
				EnableJS:     false,
				JSWaitTime:   2,
				AutoDetectJS: true, // Auto-detect by default when crawling is enabled
			},
			Network: NetworkConfig{
				EnablePortScan:  false,
				PortScanTimeout: defaultPortScanTimeoutSecs,
				Ports:           nil,
				MaxPortWorkers:  defaultPortScanWorkers,
			},
		},
	}
}

func detectOperatorFromEnv() string {
	if env := os.Getenv("USER"); env != "" {
		return env
	}
	if env := os.Getenv("LOGNAME"); env != "" {
		return env
	}
	return ""
}

func loadDefaultOverrides() defaultOverrides {
	overrides := defaultOverrides{}

	if viper.IsSet("defaults.timeout_secs") {
		val := viper.GetInt("defaults.timeout_secs")
		overrides.TimeoutSecs = &val
	}

	if viper.IsSet("defaults.telemetry") {
		val := viper.GetBool("defaults.telemetry")
		overrides.TelemetryEnabled = &val
	}

	if viper.IsSet("defaults.operator") {
		overrides.Operator = viper.GetString("defaults.operator")
		overrides.OperatorOverride = true
	}

	if viper.IsSet("defaults.retention_days") {
		val := viper.GetInt("defaults.retention_days")
		overrides.RetentionDays = &val
	}

	if viper.IsSet("defaults.hash_algorithm") {
		overrides.HashAlgorithm = viper.GetString("defaults.hash_algorithm")
	}

	if viper.IsSet("defaults.secure_results") {
		val := viper.GetBool("defaults.secure_results")
		overrides.SecureResults = &val
	}

	return overrides
}

// applyConfigDefaults merges config file defaults into the runtime config when the user
// did not explicitly override the corresponding flag.
func applyConfigDefaults(cmd *cobra.Command) {
	overrides := loadDefaultOverrides()

	if overrides.OperatorOverride && overrides.Operator != "" {
		cliConfig.Defaults.Operator = overrides.Operator
		setStringFlagIfUnset(cmd.Flags(), "operator", overrides.Operator)
	}

	if overrides.TimeoutSecs != nil {
		applyIntDefault(checkCmd.PersistentFlags(), "timeout", *overrides.TimeoutSecs, func(v int) {
			cliConfig.Defaults.TimeoutSecs = v
			cliConfig.Check.TimeoutSecs = v
		})
	}

	if overrides.TelemetryEnabled != nil {
		applyBoolDefault(checkCmd.PersistentFlags(), "telemetry", *overrides.TelemetryEnabled, func(v bool) {
			cliConfig.Defaults.TelemetryEnabled = v
			cliConfig.Check.TelemetryEnabled = v
		})
	}

	if overrides.RetentionDays != nil {
		applyIntDefault(checkHTTPCmdDDD.Flags(), "retention-days", *overrides.RetentionDays, func(v int) {
			cliConfig.Defaults.RetentionDays = v
			cliConfig.Check.RetentionDays = v
		})
	}

	if overrides.HashAlgorithm != "" {
		if algo, err := ParseHashAlgorithm(overrides.HashAlgorithm); err == nil {
			cliConfig.Check.HashAlgorithm = algo.String()
		}
	}

	if overrides.SecureResults != nil {
		cliConfig.Check.SecureResults = *overrides.SecureResults
	}
}

func applyIntDefault(flags *pflag.FlagSet, name string, value int, setter func(int)) {
	if flags == nil || setter == nil {
		return
	}
	flag := flags.Lookup(name)
	if flag != nil && flag.Changed {
		return
	}
	setter(value)
}

func applyBoolDefault(flags *pflag.FlagSet, name string, value bool, setter func(bool)) {
	if flags == nil || setter == nil {
		return
	}
	flag := flags.Lookup(name)
	if flag != nil && flag.Changed {
		return
	}
	setter(value)
}

func setStringFlagIfUnset(flags *pflag.FlagSet, name, value string) {
	if flags == nil {
		return
	}
	flag := flags.Lookup(name)
	if flag == nil || flag.Changed {
		return
	}
	_ = flag.Value.Set(value)
}
