package cmd

import (
	"fmt"

	"github.com/DanielCoulbourne/orch/tools/linear-cli/internal/api"
	"github.com/spf13/cobra"
)

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show current user info",
	Long: `Display information about the authenticated Linear user.

Examples:
  linear me
  linear me -o json`,
	RunE: runMe,
}

func init() {
	rootCmd.AddCommand(meCmd)
}

func runMe(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	output, _ := cmd.Root().PersistentFlags().GetString("output")

	query := `query {
		viewer {
			id
			name
			displayName
			email
			admin
			organization { name }
		}
	}`

	type Viewer struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		DisplayName  string `json:"displayName"`
		Email        string `json:"email"`
		Admin        bool   `json:"admin"`
		Organization struct {
			Name string `json:"name"`
		} `json:"organization"`
	}

	var result struct {
		Viewer Viewer `json:"viewer"`
	}

	if err := client.DoInto(query, nil, &result); err != nil {
		return err
	}

	if output == "json" {
		return printJSON(result.Viewer)
	}

	v := result.Viewer
	fmt.Printf("%s (%s)\n", v.DisplayName, v.Email)
	fmt.Printf("Org: %s\n", v.Organization.Name)
	return nil
}
