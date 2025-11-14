package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/domain/engagement"
	sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"
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

type engagementDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Owner     string    `json:"owner"`
	Start     time.Time `json:"start,omitempty"`
	End       time.Time `json:"end,omitempty"`
	Scope     []string  `json:"scope,omitempty"`
	ROE       string    `json:"roe,omitempty"`
	ROEAgree  bool      `json:"roe_agree"`
	CreatedAt time.Time `json:"created_at"`
}

func engagementToDTO(eng *engagement.Engagement) engagementDTO {
	return engagementDTO{
		ID:        eng.ID(),
		Name:      eng.Name(),
		Owner:     eng.Owner(),
		Start:     eng.Start(),
		End:       eng.End(),
		Scope:     eng.Scope(),
		ROE:       eng.ROE(),
		ROEAgree:  eng.ROEAgreed(),
		CreatedAt: eng.CreatedAt(),
	}
}

var engagementCmd = &cobra.Command{
	Use:   "engagement",
	Short: "Manage engagements (create/list/add-scope...)",
}

var engagementCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new engagement (requires ROE acceptance)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		name, _ := cmd.Flags().GetString("name")
		owner, _ := cmd.Flags().GetString("owner")
		roe, _ := cmd.Flags().GetString("roe")
		roeAgree, _ := cmd.Flags().GetBool("roe-agree")
		scopeFlag, _ := cmd.Flags().GetStringSlice("scope")

		if name == "" || owner == "" {
			return errors.New("name and owner are required")
		}

		if !roeAgree {
			return errors.New("ROE must be agreed (--roe-agree)")
		}

		eng, err := appCtx.Services.EngagementService.CreateEngagement(ctx, name, owner, roe, scopeFlag)
		if err != nil {
			return fmt.Errorf("failed to create engagement: %w", err)
		}

		if err := appCtx.Services.EngagementService.AcknowledgeROE(ctx, eng.ID()); err != nil {
			return fmt.Errorf("failed to acknowledge ROE: %w", err)
		}

		fmt.Printf("%s engagement %s (id=%s)\n", colorSuccess("Created"), name, eng.ID())
		return nil
	},
}

var engagementListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all engagements",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		engagements, err := appCtx.Services.EngagementService.ListEngagements(ctx)
		if err != nil {
			return fmt.Errorf("failed to list engagements: %w", err)
		}

		dtos := make([]engagementDTO, len(engagements))
		for i, eng := range engagements {
			dtos[i] = engagementToDTO(eng)
		}

		b, _ := json.MarshalIndent(dtos, jsonPrefix, jsonIndent)
		fmt.Println(string(b))
		return nil
	},
}

var engagementViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View a single engagement",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}

		eng, err := appCtx.Services.EngagementService.GetEngagement(ctx, id)
		if err != nil {
			if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
				return fmt.Errorf("engagement %s not found", id)
			}
			return fmt.Errorf("failed to get engagement: %w", err)
		}

		dto := engagementToDTO(eng)
		b, _ := json.MarshalIndent(dto, jsonPrefix, jsonIndent)
		fmt.Println(string(b))
		return nil
	},
}

var engagementAddScopeCmd = &cobra.Command{
	Use:   "add-scope",
	Short: "Add scope entries (URLs/hosts) to an engagement",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}

		scopeEntries, _ := cmd.Flags().GetStringSlice("scope")
		if len(scopeEntries) == 0 {
			return fmt.Errorf("--scope is required (one or more entries to add)")
		}

		normalized, err := normalizeScopeEntries(id, scopeEntries)
		if err != nil {
			return fmt.Errorf("invalid scope entries: %w", err)
		}

		if err := appCtx.Services.EngagementService.AddToScope(ctx, id, normalized); err != nil {
			return fmt.Errorf("failed to add scope: %w", err)
		}

		fmt.Printf("%s added %d scope entries to engagement %s\n", colorSuccess("Success:"), len(normalized), id)
		return nil
	},
}

var engagementRemoveScopeCmd = &cobra.Command{
	Use:   "remove-scope",
	Short: "Remove scope entries (URLs/hosts) from an engagement",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}

		domains, _ := cmd.Flags().GetStringSlice("domain")
		if len(domains) == 0 {
			return fmt.Errorf("--domain is required (one or more domains to remove)")
		}

		if err := appCtx.Services.EngagementService.RemoveFromScope(ctx, id, domains); err != nil {
			return fmt.Errorf("failed to remove scope: %w", err)
		}

		fmt.Printf("%s removed %d scope entries from engagement %s\n", colorSuccess("Success:"), len(domains), id)
		return nil
	},
}

var engagementDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an engagement",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		id, _ := cmd.Flags().GetString("id")
		confirm, _ := cmd.Flags().GetBool("confirm")

		if id == "" {
			return fmt.Errorf("--id is required")
		}

		if !confirm {
			return errors.New("--confirm is required to delete the engagement")
		}

		if err := appCtx.Services.EngagementService.DeleteEngagement(ctx, id); err != nil {
			return fmt.Errorf("failed to delete engagement: %w", err)
		}

		fmt.Printf("%s engagement %s deleted\n", colorSuccess("Success:"), id)
		return nil
	},
}

func init() {
	// Use DDD-based commands instead of old ones
	engagementCmd.AddCommand(engagementCreateCmd)
	engagementCmd.AddCommand(engagementListCmd)
	engagementCmd.AddCommand(engagementViewCmd)
	engagementCmd.AddCommand(engagementAddScopeCmd)
	engagementCmd.AddCommand(engagementRemoveScopeCmd)
	engagementCmd.AddCommand(engagementDeleteCmd)

	engagementCreateCmd.Flags().String("name", "", "Engagement name")
	engagementCreateCmd.Flags().String("owner", "", "Engagement owner")
	engagementCreateCmd.Flags().String("roe", "", "Rules of Engagement")
	engagementCreateCmd.Flags().Bool("roe-agree", false, "Acknowledge ROE")
	engagementCreateCmd.Flags().StringSlice("scope", nil, "Initial scope entries")

	engagementViewCmd.Flags().String("id", "", "Engagement ID")

	engagementAddScopeCmd.Flags().String("id", "", "Engagement ID")
	engagementAddScopeCmd.Flags().StringSlice("scope", nil, "Scope entries to add")

	engagementRemoveScopeCmd.Flags().String("id", "", "Engagement ID")
	engagementRemoveScopeCmd.Flags().StringSlice("domain", nil, "Domains to remove")

	engagementDeleteCmd.Flags().String("id", "", "Engagement ID")
	engagementDeleteCmd.Flags().Bool("confirm", false, "Confirm deletion")
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
// Used by: engagement commands and tui.go
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
