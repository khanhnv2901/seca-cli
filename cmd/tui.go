package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive terminal UI for engagement management",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI(getAppContext(cmd))
	},
}

func runTUI(appCtx *AppContext) error {
	if appCtx.Services == nil || appCtx.Services.EngagementService == nil {
		return fmt.Errorf("engagement services not initialized")
	}

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	for {
		dtos, err := listEngagementDTOs(ctx, appCtx)
		if err != nil {
			return err
		}

		fmt.Println("=== SECA Engagements ===")
		if len(dtos) == 0 {
			fmt.Println("No engagements found. Use `seca engagement create` to add one.")
		}
		for i, eng := range dtos {
			fmt.Printf("[%d] %s (Owner: %s, Scope: %d target(s))\n", i+1, eng.Name, eng.Owner, len(eng.Scope))
		}
		fmt.Println("[a] Add scope    [r] Refresh    [q] Quit")
		fmt.Print("Select engagement: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch strings.ToLower(input) {
		case "q":
			return nil
		case "r", "":
			continue
		case "a":
			if err := handleAddScope(ctx, reader, appCtx); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			continue
		default:
			index, err := strconv.Atoi(input)
			if err != nil || index < 1 || index > len(dtos) {
				fmt.Println("Invalid selection")
				continue
			}
			showEngagementDetail(reader, dtos[index-1])
		}
	}
}

func listEngagementDTOs(ctx context.Context, appCtx *AppContext) ([]engagementDTO, error) {
	engagements, err := appCtx.Services.EngagementService.ListEngagements(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list engagements: %w", err)
	}

	dtos := make([]engagementDTO, len(engagements))
	for i, eng := range engagements {
		dtos[i] = engagementToDTO(eng)
	}
	return dtos, nil
}

func handleAddScope(ctx context.Context, reader *bufio.Reader, appCtx *AppContext) error {
	fmt.Print("Enter engagement ID: ")
	id, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read engagement ID: %w", err)
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("ID required")
	}

	fmt.Print("Enter scope entries (comma separated): ")
	scopeLine, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read scope entries: %w", err)
	}
	scopeLine = strings.TrimSpace(scopeLine)
	if scopeLine == "" {
		return fmt.Errorf("scope required")
	}

	entries := strings.Split(scopeLine, ",")
	for i := range entries {
		entries[i] = strings.TrimSpace(entries[i])
	}

	normalized, err := normalizeScopeEntries(id, entries)
	if err != nil {
		return err
	}

	if err := appCtx.Services.EngagementService.AddToScope(ctx, id, normalized); err != nil {
		return fmt.Errorf("failed to add scope entries: %w", err)
	}

	fmt.Printf("%s scope %v to engagement %s\n", colorSuccess("Added"), normalized, id)
	return nil
}

func showEngagementDetail(reader *bufio.Reader, eng engagementDTO) {
	fmt.Println("--------------------------------------------------")
	fmt.Printf("Name  : %s\n", eng.Name)
	fmt.Printf("ID    : %s\n", eng.ID)
	fmt.Printf("Owner : %s\n", eng.Owner)
	fmt.Printf("Scope (%d):\n", len(eng.Scope))
	for _, s := range eng.Scope {
		fmt.Printf("  - %s\n", s)
	}
	fmt.Printf("Created: %s\n", eng.CreatedAt.Format("2006-01-02 15:04:05"))
	if eng.ROE != "" {
		fmt.Printf("ROE   : %s\n", eng.ROE)
	}
	fmt.Println("Press Enter to return...")
	_, _ = reader.ReadString('\n')
}
