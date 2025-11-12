package cmd

import (
	"bufio"
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
		return runTUI()
	},
}

func runTUI() error {
	reader := bufio.NewReader(os.Stdin)
	for {
		list := loadEngagements()
		fmt.Println("=== SECA Engagements ===")
		if len(list) == 0 {
			fmt.Println("No engagements found. Use `seca engagement create` to add one.")
		}
		for i, eng := range list {
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
			if err := handleAddScope(reader, list); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			continue
		default:
			index, err := strconv.Atoi(input)
			if err != nil || index < 1 || index > len(list) {
				fmt.Println("Invalid selection")
				continue
			}
			showEngagementDetail(reader, list[index-1])
		}
	}
}

func handleAddScope(reader *bufio.Reader, engagements []Engagement) error {
	if len(engagements) == 0 {
		return fmt.Errorf("no engagements available")
	}
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
	add := strings.Split(scopeLine, ",")
	for i := range add {
		add[i] = strings.TrimSpace(add[i])
	}
	return addScopeEntries(id, add)
}

func showEngagementDetail(reader *bufio.Reader, eng Engagement) {
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
