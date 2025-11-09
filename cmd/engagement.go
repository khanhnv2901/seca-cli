package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

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

// Deprecated: engagementsFile is kept for backward compatibility in tests
// Use getEngagementsFilePath() instead
const engagementsFile = "engagements.json"

var engagementCmd = &cobra.Command{
	Use:   "engagement",
	Short: "Manage engagements (create/list/add-scope...)",
}

var engagementCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new engagement (requires ROE acceptance)",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		owner, _ := cmd.Flags().GetString("owner")
		roe, _ := cmd.Flags().GetString("roe")
		roeAgree, _ := cmd.Flags().GetBool("roe-agree")
		if name == "" || owner == "" {
			return errors.New("name and owner are required")
		}
		if !roeAgree {
			return errors.New("ROE must be aggred (--roe-agree)")
		}
		e := Engagement{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			Name:      name,
			Owner:     owner,
			ROE:       roe,
			ROEAgree:  roeAgree,
			CreatedAt: time.Now(),
		}
		list := loadEngagements()
		list = append(list, e)
		saveEngagements(list)
		fmt.Printf("Created engagement %s (id=%s)\n", name, e.ID)
		return nil
	},
}

var engagementListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all engagements",
	Run: func(cmd *cobra.Command, args []string) {
		list := loadEngagements()
		b, _ := json.MarshalIndent(list, "", "  ")
		fmt.Println(string(b))
	},
}

var engagementAddScopeCmd = &cobra.Command{
	Use:   "add-scope",
	Short: "Add scope entries (URLs/hosts) to an engagement",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}
		add, _ := cmd.Flags().GetStringSlice("scope")
		if len(add) == 0 {
			return errors.New("--scope must contain one or more hosts/urls")
		}

		list := loadEngagements()
		found := false
		for i := range list {
			if list[i].ID == id {
				list[i].Scope = append(list[i].Scope, add...)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no engagement found with id %s", id)
		}
		saveEngagements(list)
		fmt.Printf("Added scope %s to engagement %s\n", add, id)
		return nil
	},
}

func init() {
	engagementCreateCmd.Flags().String("name", "", "Engagement name")
	engagementCreateCmd.Flags().String("owner", "", "Onwer/POC")
	engagementCreateCmd.Flags().String("roe", "", "Rules of engagement text (or path)")
	engagementCreateCmd.Flags().Bool("roe-agree", false, "I confirm ROE and explicit authorization for this engagement")
	engagementCmd.AddCommand(engagementCreateCmd)
	engagementCmd.AddCommand(engagementListCmd)
	engagementAddScopeCmd.Flags().String("id", "", "Engagement id")
	engagementAddScopeCmd.Flags().StringSlice("scope", []string{}, "Scope entries (URLs/hosts)")
	engagementCmd.AddCommand(engagementAddScopeCmd)
}

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

func saveEngagements(list []Engagement) {
	filePath, err := getEngagementsFilePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting engagements file path: %v\n", err)
		return
	}

	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling engagements: %v\n", err)
		return
	}

	if err := os.WriteFile(filePath, b, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing engagements file: %v\n", err)
	}
}
