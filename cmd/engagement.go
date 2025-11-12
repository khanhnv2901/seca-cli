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

	consts "github.com/khanhnv2901/seca-cli/internal/constants"
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
		fmt.Printf("%s engagement %s (id=%s)\n", colorSuccess("Created"), name, e.ID)
		return nil
	},
}

var engagementListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all engagements",
	Run: func(cmd *cobra.Command, args []string) {
		list := loadEngagements()
		b, _ := json.MarshalIndent(list, jsonPrefix, jsonIndent)
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
		return addScopeEntries(id, add)
	},
}

var engagementViewCmd = &cobra.Command{
	Use:   "view",
	Short: "Show a single engagement as JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}

		eng, err := findEngagementByID(id)
		if err != nil {
			return err
		}

		payload, err := json.MarshalIndent(eng, jsonPrefix, jsonIndent)
		if err != nil {
			return err
		}

		fmt.Println(string(payload))
		return nil
	},
}

var engagementDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an engagement from tracking",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}

		list := loadEngagements()
		index := -1
		for i := range list {
			if list[i].ID == id {
				index = i
				break
			}
		}

		if index == -1 {
			return &EngagementNotFoundError{ID: id}
		}

		removed := list[index]
		list = append(list[:index], list[index+1:]...)
		saveEngagements(list)
		fmt.Printf("%s engagement %s (%s) removed\n", colorWarn("Deleted"), removed.Name, removed.ID)
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
	engagementViewCmd.Flags().String("id", "", "Engagement id")
	engagementCmd.AddCommand(engagementViewCmd)
	engagementDeleteCmd.Flags().String("id", "", "Engagement id")
	engagementCmd.AddCommand(engagementDeleteCmd)
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

	b, err := json.MarshalIndent(list, jsonPrefix, jsonIndent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling engagements: %v\n", err)
		return
	}

	if err := os.WriteFile(filePath, b, consts.DefaultFilePerm); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing engagements file: %v\n", err)
	}
}

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

func addScopeEntries(id string, entries []string) error {
	if id == "" {
		return errors.New("--id is required")
	}
	if len(entries) == 0 {
		return errors.New("--scope must contain one or more hosts/urls")
	}

	normalizedScope, err := normalizeScopeEntries(id, entries)
	if err != nil {
		return err
	}

	list := loadEngagements()
	found := false
	for i := range list {
		if list[i].ID == id {
			list[i].Scope = append(list[i].Scope, normalizedScope...)
			found = true
			break
		}
	}
	if !found {
		return &EngagementNotFoundError{ID: id}
	}
	saveEngagements(list)
	fmt.Printf("%s scope %v to engagement %s\n", colorSuccess("Added"), normalizedScope, id)
	return nil
}

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
