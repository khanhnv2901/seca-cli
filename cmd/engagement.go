package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	consts "github.com/khanhnv2901/seca-cli/internal/shared/constants"
	"github.com/spf13/cobra"
)

type Engagement struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Owner     string    `json:"owner"`
	Start     time.Time `json:"start,omitempty"`
	End       time.Time `json:"end,omitempty"`
	Scope     []string  `json:"scope,omitempty"` // list of allowed hosts/urls
	ROE       string    `json:"roe,omitempty"`   // rules of engagement text
	ROEAgree  bool      `json:"roe_agree"`
	CreatedAt time.Time `json:"created_at"`
}

var engagementCmd = &cobra.Command{
	Use:   "engagement",
	Short: "Manage engagements (create/list/add-scope...)",
}

// Old command implementations removed - now using DDD-based commands from engagement_ddd.go
// The old engagementCreateCmd, engagementListCmd, engagementViewCmd, engagementAddScopeCmd,
// engagementRemoveScopeCmd, and engagementDeleteCmd have been replaced with DDD versions.

func init() {
	// Use DDD-based commands instead of old ones
	engagementCmd.AddCommand(engagementCreateCmdDDD)
	engagementCmd.AddCommand(engagementListCmdDDD)
	engagementCmd.AddCommand(engagementViewCmdDDD)
	engagementCmd.AddCommand(engagementAddScopeCmdDDD)
	engagementCmd.AddCommand(engagementRemoveScopeCmdDDD)
	engagementCmd.AddCommand(engagementDeleteCmdDDD)
}

// ============================================================================
// Legacy Helper Functions
// ============================================================================
// These functions are kept for backward compatibility with tests, TUI, and
// parts of serve.go that haven't been fully migrated yet.
// TODO: Migrate remaining usages to use application services via AppContext
// ============================================================================

// loadEngagements loads all engagements from the JSON file.
// DEPRECATED: Use appCtx.Services.EngagementService.ListEngagements() instead.
// Still used by: engagement_test.go, tui.go, check.go
func loadEngagements() []Engagement {
	filePath, err := getEngagementsFilePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting engagements file path: %v\n", err)
		return []Engagement{}
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []Engagement{}
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading engagements file: %v\n", err)
		return []Engagement{}
	}

	var out []Engagement
	if err := json.Unmarshal(b, &out); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing engagements file: %v\n", err)
		return []Engagement{}
	}

	return out
}

// saveEngagements saves engagements to the JSON file.
// DEPRECATED: Use appCtx.Services.EngagementService methods instead.
// Still used by: engagement_test.go, info_test.go
func saveEngagements(list []Engagement) {
	filePath, err := getEngagementsFilePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting engagements file path: %v\n", err)
		return
	}

	b, err := json.MarshalIndent(list, jsonPrefix, jsonIndent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling engagements: %v\n", err)
		return
	}

	if err := os.WriteFile(filePath, b, consts.DefaultFilePerm); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing engagements file: %v\n", err)
	}
}

// findEngagementByID finds an engagement by ID.
// DEPRECATED: Use appCtx.Services.EngagementService.GetEngagement() instead.
// Still used by: serve.go, engagement_test.go
func findEngagementByID(id string) (*Engagement, error) {
	if id == "" {
		return nil, fmt.Errorf("--id is required")
	}
	list := loadEngagements()
	for i := range list {
		if list[i].ID == id {
			eng := list[i]
			return &eng, nil
		}
	}
	return nil, &EngagementNotFoundError{ID: id}
}

var allowedScopeSchemes = map[string]struct{}{
	"http":  {},
	"https": {},
}

// ============================================================================
// Scope Management Functions
// ============================================================================
// These functions are used by TUI and DDD commands for scope management.
// ============================================================================

// normalizeScopeEntries validates and normalizes scope entries.
// Used by: engagement_ddd.go (DDD commands), tui.go
func normalizeScopeEntries(scopeID string, entries []string) ([]string, error) {
	out := make([]string, len(entries))
	for i, entry := range entries {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			baseErr := &ScopeViolationError{
				Target: entry,
				Scope:  scopeID,
			}
			return nil, fmt.Errorf("%w: entry is empty or whitespace", baseErr)
		}
		if err := validateScopeEntry(trimmed); err != nil {
			baseErr := &ScopeViolationError{
				Target: trimmed,
				Scope:  scopeID,
			}
			return nil, fmt.Errorf("%w: %v", baseErr, err)
		}
		out[i] = trimmed
	}
	return out, nil
}

func validateScopeEntry(entry string) error {
	if strings.Contains(entry, "://") {
		return validateURLScope(entry)
	}

	if net.ParseIP(entry) != nil {
		return nil
	}

	if parsed, err := url.Parse("http://" + entry); err == nil && parsed.Host != "" {
		host := parsed.Hostname()
		if host == "" {
			return errors.New("missing hostname")
		}
		if !isValidHostOrIP(host) {
			return fmt.Errorf("invalid host %q", host)
		}
		return nil
	}

	return errors.New("must be a valid http(s) URL or hostname/IP")
}

func validateURLScope(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme == "" || u.Host == "" {
		return errors.New("URL must include a scheme and host")
	}

	if _, ok := allowedScopeSchemes[strings.ToLower(u.Scheme)]; !ok {
		return fmt.Errorf("unsupported scheme %q (only http/https are allowed)", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return errors.New("URL must include a hostname")
	}

	if !isValidHostOrIP(host) {
		return fmt.Errorf("invalid host %q", host)
	}

	return nil
}

func isValidHostOrIP(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}
	return isValidHostname(host)
}

func isValidHostname(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}

	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return false
	}

	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, ch := range label {
			if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-') {
				return false
			}
		}
	}
	return true
}
