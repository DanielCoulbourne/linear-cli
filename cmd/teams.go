package cmd

import (
	"fmt"

	"github.com/DanielCoulbourne/orch/tools/linear-cli/internal/api"
	"github.com/spf13/cobra"
)

var teamsCmd = &cobra.Command{
	Use:   "teams",
	Short: "List teams",
	Long: `List all teams in your Linear workspace.

Examples:
  linear teams
  linear teams -o json`,
	RunE: runTeams,
}

func init() {
	rootCmd.AddCommand(teamsCmd)
}

func runTeams(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	output, _ := cmd.Root().PersistentFlags().GetString("output")

	query := `query {
		teams {
			nodes {
				key
				name
				description
				members { nodes { displayName } }
			}
		}
	}`

	type Team struct {
		Key         string `json:"key"`
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Members     struct {
			Nodes []struct {
				DisplayName string `json:"displayName"`
			} `json:"nodes"`
		} `json:"members"`
	}

	var result struct {
		Teams struct {
			Nodes []Team `json:"nodes"`
		} `json:"teams"`
	}

	if err := client.DoInto(query, nil, &result); err != nil {
		return err
	}

	if output == "json" {
		return printJSON(result.Teams.Nodes)
	}

	for _, t := range result.Teams.Nodes {
		fmt.Printf("%-8s %s (%d members)\n", t.Key, t.Name, len(t.Members.Nodes))
	}
	return nil
}
