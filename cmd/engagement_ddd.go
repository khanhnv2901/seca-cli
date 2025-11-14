package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/domain/engagement"
	sharedErrors "github.com/khanhnv2901/seca-cli/internal/shared/errors"
	"github.com/spf13/cobra"
)

// engagementDTO is used for JSON output to maintain backward compatibility
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

// toDTO converts domain entity to DTO for JSON output
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

// DDD-based engagement commands

var engagementCreateCmdDDD = &cobra.Command{
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

		// Use EngagementService to create
		eng, err := appCtx.Services.EngagementService.CreateEngagement(ctx, name, owner, roe, scopeFlag)
		if err != nil {
			return fmt.Errorf("failed to create engagement: %w", err)
		}

		// Acknowledge ROE since the flag was set
		if err := appCtx.Services.EngagementService.AcknowledgeROE(ctx, eng.ID()); err != nil {
			return fmt.Errorf("failed to acknowledge ROE: %w", err)
		}

		fmt.Printf("%s engagement %s (id=%s)\n", colorSuccess("Created"), name, eng.ID())
		return nil
	},
}

var engagementListCmdDDD = &cobra.Command{
	Use:   "list",
	Short: "List all engagements",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		engagements, err := appCtx.Services.EngagementService.ListEngagements(ctx)
		if err != nil {
			return fmt.Errorf("failed to list engagements: %w", err)
		}

		// Convert to DTOs for JSON output
		dtos := make([]engagementDTO, len(engagements))
		for i, eng := range engagements {
			dtos[i] = engagementToDTO(eng)
		}

		b, _ := json.MarshalIndent(dtos, jsonPrefix, jsonIndent)
		fmt.Println(string(b))
		return nil
	},
}

var engagementViewCmdDDD = &cobra.Command{
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

var engagementAddScopeCmdDDD = &cobra.Command{
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

		// Normalize scope entries (using existing validation)
		normalized, err := normalizeScopeEntries(id, scopeEntries)
		if err != nil {
			return fmt.Errorf("invalid scope entries: %w", err)
		}

		// Add to scope using service
		if err := appCtx.Services.EngagementService.AddToScope(ctx, id, normalized); err != nil {
			return fmt.Errorf("failed to add scope: %w", err)
		}

		fmt.Printf("%s added %d scope entries to engagement %s\n", colorSuccess("Success:"), len(normalized), id)
		return nil
	},
}

var engagementRemoveScopeCmdDDD = &cobra.Command{
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

		// Remove from scope using service
		if err := appCtx.Services.EngagementService.RemoveFromScope(ctx, id, domains); err != nil {
			return fmt.Errorf("failed to remove scope: %w", err)
		}

		fmt.Printf("%s removed %d scope entries from engagement %s\n", colorSuccess("Success:"), len(domains), id)
		return nil
	},
}

var engagementDeleteCmdDDD = &cobra.Command{
	Use:   "delete",
	Short: "Delete an engagement",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		appCtx := getAppContext(cmd)

		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}

		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			return fmt.Errorf("must pass --confirm to delete engagement")
		}

		if err := appCtx.Services.EngagementService.DeleteEngagement(ctx, id); err != nil {
			if errors.Is(err, sharedErrors.ErrEngagementNotFound) {
				return fmt.Errorf("engagement %s not found", id)
			}
			return fmt.Errorf("failed to delete engagement: %w", err)
		}

		fmt.Printf("%s deleted engagement %s\n", colorSuccess("Success:"), id)
		return nil
	},
}

func init() {
	// Keep the old commands for now, but we can add new DDD-based ones
	// When ready, we can replace the old ones

	// Create flags
	engagementCreateCmdDDD.Flags().String("name", "", "Engagement name")
	engagementCreateCmdDDD.Flags().String("owner", "", "Engagement owner")
	engagementCreateCmdDDD.Flags().String("roe", "", "Rules of Engagement")
	engagementCreateCmdDDD.Flags().Bool("roe-agree", false, "Acknowledge ROE")
	engagementCreateCmdDDD.Flags().StringSlice("scope", nil, "Initial scope entries")

	// View flags
	engagementViewCmdDDD.Flags().String("id", "", "Engagement ID")

	// Add scope flags
	engagementAddScopeCmdDDD.Flags().String("id", "", "Engagement ID")
	engagementAddScopeCmdDDD.Flags().StringSlice("scope", nil, "Scope entries to add")

	// Remove scope flags
	engagementRemoveScopeCmdDDD.Flags().String("id", "", "Engagement ID")
	engagementRemoveScopeCmdDDD.Flags().StringSlice("domain", nil, "Domains to remove")

	// Delete flags
	engagementDeleteCmdDDD.Flags().String("id", "", "Engagement ID")
	engagementDeleteCmdDDD.Flags().Bool("confirm", false, "Confirm deletion")
}
