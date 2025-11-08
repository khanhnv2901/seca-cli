package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate report for an engagement (from results)",
}

var reportGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate JSON report for an engagement id",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}

		// read results from a results directory (this is app specific)
		path := fmt.Sprintf("results_%s.json", id)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("no results found at %s", path)
		}
		b, _ := os.ReadFile(path)
		fmt.Printf("Report for engagement %s:\n", id)
		// pretty print
		var pretty map[string]interface{}
		_ = json.Unmarshal(b, &pretty)
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
		// optionally create markdown summary
		md := fmt.Sprintf("# Engagement %s Report\nGenerated: %s\n\n(see JSON for details)\n", id, time.Now().Format(time.RFC3339))
		_ = os.WriteFile(fmt.Sprintf("report_%s.md", id), []byte(md), 0644)

		return nil
	},
}

func init() {
	reportGenerateCmd.Flags().String("id", "", "Engagement id")
	reportCmd.AddCommand(reportGenerateCmd)
}
