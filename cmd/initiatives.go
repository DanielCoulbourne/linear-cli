package cmd

import (
	"fmt"
	"strings"

	"github.com/DanielCoulbourne/orch/tools/linear-cli/internal/api"
	"github.com/spf13/cobra"
)

var initiativesCmd = &cobra.Command{
	Use:   "initiatives",
	Short: "Manage Linear initiatives",
	Long:  `List, view, create, and update initiatives in Linear.`,
}

var initiativesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List initiatives",
	Long: `List initiatives with optional filters.

Examples:
  linear initiatives list
  linear initiatives list --status Active
  linear initiatives list -o json`,
	RunE: runInitiativesList,
}

var initiativesViewCmd = &cobra.Command{
	Use:   "view [ID]",
	Short: "View a single initiative",
	Long: `View details of a specific initiative by its slug or ID.

Examples:
  linear initiatives view <slug-id>
  linear initiatives view <slug-id> -o json`,
	Args: cobra.ExactArgs(1),
	RunE: runInitiativesView,
}

var initiativesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new initiative",
	Long: `Create a new initiative in Linear.

Statuses: Planned, Active, Completed

Examples:
  linear initiatives create --name "Q2 Platform Rebuild"
  linear initiatives create --name "Launch v2" --description "Ship the new version" --status Active
  linear initiatives create --name "Infra Overhaul" --target-date 2026-06`,
	RunE: runInitiativesCreate,
}

var initiativesUpdateCmd = &cobra.Command{
	Use:   "update [ID]",
	Short: "Update an existing initiative",
	Long: `Update fields on an existing initiative.

Examples:
  linear initiatives update <id> --status Active
  linear initiatives update <id> --name "Renamed Initiative"`,
	Args: cobra.ExactArgs(1),
	RunE: runInitiativesUpdate,
}

var initiativesLinkProjectCmd = &cobra.Command{
	Use:   "link-project",
	Short: "Link a project to an initiative",
	Long: `Associate a project with an initiative.

Examples:
  linear initiatives link-project --initiative "Q2 Platform Rebuild" --project "Auth Rewrite"`,
	RunE: runInitiativesLinkProject,
}

var initiativesUnlinkProjectCmd = &cobra.Command{
	Use:   "unlink-project",
	Short: "Unlink a project from an initiative",
	Long: `Remove the association between a project and an initiative.

Examples:
  linear initiatives unlink-project --initiative "Q2 Platform Rebuild" --project "Auth Rewrite"`,
	RunE: runInitiativesUnlinkProject,
}

func init() {
	rootCmd.AddCommand(initiativesCmd)
	initiativesCmd.AddCommand(initiativesListCmd)
	initiativesCmd.AddCommand(initiativesViewCmd)
	initiativesCmd.AddCommand(initiativesCreateCmd)
	initiativesCmd.AddCommand(initiativesUpdateCmd)
	initiativesCmd.AddCommand(initiativesLinkProjectCmd)
	initiativesCmd.AddCommand(initiativesUnlinkProjectCmd)

	// list flags
	initiativesListCmd.Flags().String("status", "", "Filter by status: Planned, Active, Completed")

	// create flags
	initiativesCreateCmd.Flags().String("name", "", "Initiative name (required)")
	initiativesCreateCmd.Flags().String("description", "", "Initiative description")
	initiativesCreateCmd.Flags().String("status", "", "Status: Planned, Active, Completed")
	initiativesCreateCmd.Flags().String("target-date", "", "Target date (YYYY-MM, YYYY-MM-DD, or YYYY)")
	initiativesCreateCmd.MarkFlagRequired("name")

	// update flags
	initiativesUpdateCmd.Flags().String("name", "", "New name")
	initiativesUpdateCmd.Flags().String("description", "", "New description")
	initiativesUpdateCmd.Flags().String("status", "", "New status: Planned, Active, Completed")
	initiativesUpdateCmd.Flags().String("target-date", "", "New target date")

	// link-project flags
	initiativesLinkProjectCmd.Flags().String("initiative", "", "Initiative name (required)")
	initiativesLinkProjectCmd.Flags().String("project", "", "Project name (required)")
	initiativesLinkProjectCmd.MarkFlagRequired("initiative")
	initiativesLinkProjectCmd.MarkFlagRequired("project")

	// unlink-project flags
	initiativesUnlinkProjectCmd.Flags().String("initiative", "", "Initiative name (required)")
	initiativesUnlinkProjectCmd.Flags().String("project", "", "Project name (required)")
	initiativesUnlinkProjectCmd.MarkFlagRequired("initiative")
	initiativesUnlinkProjectCmd.MarkFlagRequired("project")
}

type Initiative struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Description     string       `json:"description,omitempty"`
	Status          string       `json:"status"`
	Health          string       `json:"health,omitempty"`
	Owner           *User        `json:"owner,omitempty"`
	TargetDate      string       `json:"targetDate,omitempty"`
	Projects        *ProjectList `json:"projects,omitempty"`
	ParentInitiative *InitiativeRef `json:"parentInitiative,omitempty"`
	URL             string       `json:"url,omitempty"`
	CreatedAt       string       `json:"createdAt,omitempty"`
	UpdatedAt       string       `json:"updatedAt,omitempty"`
}

type InitiativeRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProjectList struct {
	Nodes []ProjectNode `json:"nodes"`
}

type ProjectNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func runInitiativesList(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	status, _ := cmd.Flags().GetString("status")
	output, _ := cmd.Root().PersistentFlags().GetString("output")

	filterStr := ""
	if status != "" {
		filterStr = fmt.Sprintf(`filter: { status: { eq: "%s" } }`, status)
	}

	query := fmt.Sprintf(`query {
		initiatives(%s) {
			nodes {
				id
				name
				status
				health
				owner { displayName }
				targetDate
				projects { nodes { id name } }
				parentInitiative { id name }
				url
			}
		}
	}`, filterStr)

	var result struct {
		Initiatives struct {
			Nodes []Initiative `json:"nodes"`
		} `json:"initiatives"`
	}

	if err := client.DoInto(query, nil, &result); err != nil {
		return err
	}

	if output == "json" {
		return printJSON(result.Initiatives.Nodes)
	}

	if len(result.Initiatives.Nodes) == 0 {
		fmt.Println("No initiatives found.")
		return nil
	}

	for _, i := range result.Initiatives.Nodes {
		ownerName := "-"
		if i.Owner != nil {
			ownerName = i.Owner.DisplayName
		}
		projectCount := 0
		if i.Projects != nil {
			projectCount = len(i.Projects.Nodes)
		}
		healthStr := ""
		if i.Health != "" {
			healthStr = fmt.Sprintf(" [%s]", i.Health)
		}
		fmt.Printf("%-40s %-10s %-15s %d projects%s\n",
			i.Name, i.Status, ownerName, projectCount, healthStr)
	}
	return nil
}

func runInitiativesView(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	output, _ := cmd.Root().PersistentFlags().GetString("output")

	// Try finding by slug first, fall back to direct ID
	initiative, err := resolveInitiative(client, args[0])
	if err != nil {
		return err
	}

	if output == "json" {
		return printJSON(initiative)
	}

	fmt.Printf("%s\n", initiative.Name)
	fmt.Printf("Status:      %s\n", initiative.Status)
	if initiative.Health != "" {
		fmt.Printf("Health:      %s\n", initiative.Health)
	}
	if initiative.Owner != nil {
		fmt.Printf("Owner:       %s\n", initiative.Owner.DisplayName)
	}
	if initiative.TargetDate != "" {
		fmt.Printf("Target Date: %s\n", initiative.TargetDate)
	}
	if initiative.ParentInitiative != nil {
		fmt.Printf("Parent:      %s\n", initiative.ParentInitiative.Name)
	}
	if initiative.Projects != nil && len(initiative.Projects.Nodes) > 0 {
		fmt.Println("Projects:")
		for _, p := range initiative.Projects.Nodes {
			fmt.Printf("  - %s\n", p.Name)
		}
	}
	if initiative.Description != "" {
		fmt.Printf("\n%s\n", initiative.Description)
	}
	fmt.Printf("\n%s\n", initiative.URL)
	return nil
}

func runInitiativesCreate(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	status, _ := cmd.Flags().GetString("status")
	targetDate, _ := cmd.Flags().GetString("target-date")
	output, _ := cmd.Root().PersistentFlags().GetString("output")

	vars := map[string]any{
		"name": name,
	}
	if description != "" {
		vars["description"] = description
	}
	if status != "" {
		vars["status"] = status
	}
	if targetDate != "" {
		vars["targetDate"] = targetDate
	}

	query := `mutation($input: InitiativeCreateInput!) {
		initiativeCreate(input: $input) {
			initiative {
				id
				name
				status
				url
			}
		}
	}`

	var result struct {
		InitiativeCreate struct {
			Initiative Initiative `json:"initiative"`
		} `json:"initiativeCreate"`
	}

	if err := client.DoInto(query, map[string]any{"input": vars}, &result); err != nil {
		return err
	}

	i := result.InitiativeCreate.Initiative
	if output == "json" {
		return printJSON(i)
	}

	fmt.Printf("Created initiative: %s [%s]\n%s\n", i.Name, i.Status, i.URL)
	return nil
}

func runInitiativesUpdate(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	output, _ := cmd.Root().PersistentFlags().GetString("output")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	status, _ := cmd.Flags().GetString("status")
	targetDate, _ := cmd.Flags().GetString("target-date")

	// Resolve initiative by name or ID
	initiativeID, err := resolveInitiativeID(client, args[0])
	if err != nil {
		return err
	}

	vars := map[string]any{}
	if name != "" {
		vars["name"] = name
	}
	if description != "" {
		vars["description"] = description
	}
	if status != "" {
		vars["status"] = status
	}
	if targetDate != "" {
		vars["targetDate"] = targetDate
	}

	if len(vars) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query := `mutation($id: String!, $input: InitiativeUpdateInput!) {
		initiativeUpdate(id: $id, input: $input) {
			initiative {
				id
				name
				status
				url
			}
		}
	}`

	var result struct {
		InitiativeUpdate struct {
			Initiative Initiative `json:"initiative"`
		} `json:"initiativeUpdate"`
	}

	if err := client.DoInto(query, map[string]any{"id": initiativeID, "input": vars}, &result); err != nil {
		return err
	}

	i := result.InitiativeUpdate.Initiative
	if output == "json" {
		return printJSON(i)
	}

	fmt.Printf("Updated initiative: %s [%s]\n%s\n", i.Name, i.Status, i.URL)
	return nil
}

func runInitiativesLinkProject(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	initiativeName, _ := cmd.Flags().GetString("initiative")
	projectName, _ := cmd.Flags().GetString("project")

	initiativeID, err := resolveInitiativeID(client, initiativeName)
	if err != nil {
		return err
	}
	projectID, err := resolveProjectID(client, projectName)
	if err != nil {
		return err
	}

	query := `mutation($input: InitiativeToProjectCreateInput!) {
		initiativeToProjectCreate(input: $input) {
			initiativeToProject {
				initiative { name }
				project { name }
			}
		}
	}`

	var result struct {
		InitiativeToProjectCreate struct {
			InitiativeToProject struct {
				Initiative struct{ Name string } `json:"initiative"`
				Project    struct{ Name string } `json:"project"`
			} `json:"initiativeToProject"`
		} `json:"initiativeToProjectCreate"`
	}

	input := map[string]any{
		"initiativeId": initiativeID,
		"projectId":    projectID,
	}

	if err := client.DoInto(query, map[string]any{"input": input}, &result); err != nil {
		return err
	}

	r := result.InitiativeToProjectCreate.InitiativeToProject
	fmt.Printf("Linked project %q to initiative %q\n", r.Project.Name, r.Initiative.Name)
	return nil
}

func runInitiativesUnlinkProject(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	initiativeName, _ := cmd.Flags().GetString("initiative")
	projectName, _ := cmd.Flags().GetString("project")

	// Find the initiativeToProject join ID
	initiativeID, err := resolveInitiativeID(client, initiativeName)
	if err != nil {
		return err
	}
	projectID, err := resolveProjectID(client, projectName)
	if err != nil {
		return err
	}

	// Query the initiative's projects to find the join record
	query := `query($id: String!) {
		initiative(id: $id) {
			projects {
				nodes { id name }
			}
		}
	}`

	var findResult struct {
		Initiative struct {
			Projects struct {
				Nodes []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"projects"`
		} `json:"initiative"`
	}

	if err := client.DoInto(query, map[string]any{"id": initiativeID}, &findResult); err != nil {
		return err
	}

	found := false
	for _, p := range findResult.Initiative.Projects.Nodes {
		if p.ID == projectID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("project %q is not linked to initiative %q", projectName, initiativeName)
	}

	// Use the initiativeToProjectDelete mutation
	deleteQuery := `mutation($initiativeId: String!, $projectId: String!) {
		initiativeToProjectDelete(initiativeId: $initiativeId, projectId: $projectId) {
			success
		}
	}`

	var deleteResult struct {
		InitiativeToProjectDelete struct {
			Success bool `json:"success"`
		} `json:"initiativeToProjectDelete"`
	}

	if err := client.DoInto(deleteQuery, map[string]any{
		"initiativeId": initiativeID,
		"projectId":    projectID,
	}, &deleteResult); err != nil {
		return err
	}

	fmt.Printf("Unlinked project %q from initiative %q\n", projectName, initiativeName)
	return nil
}

func resolveInitiative(client *api.Client, nameOrID string) (*Initiative, error) {
	// Try by name first
	query := fmt.Sprintf(`query {
		initiatives(filter: { name: { containsIgnoreCase: "%s" } }) {
			nodes {
				id
				name
				description
				status
				health
				owner { displayName }
				targetDate
				projects { nodes { id name } }
				parentInitiative { id name }
				url
				createdAt
				updatedAt
			}
		}
	}`, nameOrID)

	var result struct {
		Initiatives struct {
			Nodes []Initiative `json:"nodes"`
		} `json:"initiatives"`
	}

	if err := client.DoInto(query, nil, &result); err != nil {
		return nil, err
	}

	if len(result.Initiatives.Nodes) == 0 {
		return nil, fmt.Errorf("initiative %q not found", nameOrID)
	}

	// Prefer exact match
	for _, i := range result.Initiatives.Nodes {
		if strings.EqualFold(i.Name, nameOrID) {
			return &i, nil
		}
	}
	return &result.Initiatives.Nodes[0], nil
}

func resolveInitiativeID(client *api.Client, nameOrID string) (string, error) {
	i, err := resolveInitiative(client, nameOrID)
	if err != nil {
		return "", err
	}
	return i.ID, nil
}
