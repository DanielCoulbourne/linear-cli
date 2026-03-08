package cmd

import (
	"fmt"

	"github.com/DanielCoulbourne/orch/tools/linear-cli/internal/api"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects",
	Long: `List all projects in your Linear workspace.

Examples:
  linear projects
  linear projects -o json`,
	RunE: runProjects,
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	output, _ := cmd.Root().PersistentFlags().GetString("output")

	query := `query {
		projects {
			nodes {
				id
				name
				state
				progress
				teams { nodes { key } }
			}
		}
	}`

	type ProjectNode struct {
		ID       string  `json:"id"`
		Name     string  `json:"name"`
		State    string  `json:"state"`
		Progress float64 `json:"progress"`
		Teams    struct {
			Nodes []struct {
				Key string `json:"key"`
			} `json:"nodes"`
		} `json:"teams"`
	}

	var result struct {
		Projects struct {
			Nodes []ProjectNode `json:"nodes"`
		} `json:"projects"`
	}

	if err := client.DoInto(query, nil, &result); err != nil {
		return err
	}

	if output == "json" {
		return printJSON(result.Projects.Nodes)
	}

	if len(result.Projects.Nodes) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	for _, p := range result.Projects.Nodes {
		var teamKeys []string
		for _, t := range p.Teams.Nodes {
			teamKeys = append(teamKeys, t.Key)
		}
		teams := "-"
		if len(teamKeys) > 0 {
			teams = fmt.Sprintf("[%s]", joinStrings(teamKeys, ", "))
		}
		fmt.Printf("%-40s %-12s %3.0f%%  %s\n", p.Name, p.State, p.Progress*100, teams)
	}
	return nil
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
