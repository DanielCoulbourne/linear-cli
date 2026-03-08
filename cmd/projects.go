package cmd

import (
	"fmt"
	"strings"

	"github.com/DanielCoulbourne/orch/tools/linear-cli/internal/api"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage Linear projects",
	Long:  `List, view, create, and update projects in Linear.`,
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	Long: `List all projects in your Linear workspace.

Examples:
  linear projects list
  linear projects list --team ENG
  linear projects list --status planned
  linear projects list -o json`,
	RunE: runProjectsList,
}

var projectsViewCmd = &cobra.Command{
	Use:   "view [name]",
	Short: "View a single project",
	Long: `View details of a specific project by name.

Examples:
  linear projects view "My Project"
  linear projects view "My Project" -o json`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectsView,
}

var projectsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	Long: `Create a new project in Linear.

Examples:
  linear projects create --name "Auth Rewrite" --team ENG
  linear projects create --name "Q2 Sprint" --team ENG --description "Sprint goals" --target-date 2026-06
  linear projects create --name "New Feature" --team ENG --initiative "Q2 Platform Rebuild"`,
	RunE: runProjectsCreate,
}

var projectsUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update an existing project",
	Long: `Update fields on an existing project.

Examples:
  linear projects update "My Project" --description "Updated desc"
  linear projects update "My Project" --target-date 2026-09
  linear projects update "My Project" --lead-me`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectsUpdate,
}

func init() {
	rootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsViewCmd)
	projectsCmd.AddCommand(projectsCreateCmd)
	projectsCmd.AddCommand(projectsUpdateCmd)

	// list flags
	projectsListCmd.Flags().String("team", "", "Filter by team key")

	// create flags
	projectsCreateCmd.Flags().String("name", "", "Project name (required)")
	projectsCreateCmd.Flags().String("team", "", "Team key (required, e.g. ENG)")
	projectsCreateCmd.Flags().String("description", "", "Project description")
	projectsCreateCmd.Flags().String("target-date", "", "Target date (YYYY-MM-DD, YYYY-MM, or YYYY)")
	projectsCreateCmd.Flags().String("start-date", "", "Start date")
	projectsCreateCmd.Flags().Bool("lead-me", false, "Set yourself as project lead")
	projectsCreateCmd.Flags().String("initiative", "", "Link to an initiative by name")
	projectsCreateCmd.Flags().Int("priority", 0, "Priority: 0=none, 1=urgent, 2=high, 3=medium, 4=low")
	projectsCreateCmd.MarkFlagRequired("name")
	projectsCreateCmd.MarkFlagRequired("team")

	// update flags
	projectsUpdateCmd.Flags().String("name", "", "New name")
	projectsUpdateCmd.Flags().String("description", "", "New description")
	projectsUpdateCmd.Flags().String("target-date", "", "New target date")
	projectsUpdateCmd.Flags().String("start-date", "", "New start date")
	projectsUpdateCmd.Flags().Bool("lead-me", false, "Set yourself as project lead")
	projectsUpdateCmd.Flags().Int("priority", -1, "New priority")
}

type ProjectFull struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	State       string  `json:"state,omitempty"`
	Progress    float64 `json:"progress"`
	Priority    int     `json:"priority"`
	StartDate   string  `json:"startDate,omitempty"`
	TargetDate  string  `json:"targetDate,omitempty"`
	URL         string  `json:"url,omitempty"`
	Lead        *User   `json:"lead,omitempty"`
	Teams       *struct {
		Nodes []struct {
			Key  string `json:"key"`
			Name string `json:"name,omitempty"`
		} `json:"nodes"`
	} `json:"teams,omitempty"`
	Initiatives *struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"initiatives,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

func runProjectsList(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	output, _ := cmd.Root().PersistentFlags().GetString("output")
	team, _ := cmd.Flags().GetString("team")

	filterClause := ""
	if team != "" {
		teamID, err := resolveTeamID(client, team)
		if err != nil {
			return err
		}
		filterClause = fmt.Sprintf(`(filter: { members: { id: { eq: "%s" } } })`, teamID)
	}

	query := fmt.Sprintf(`query {
		projects%s {
			nodes {
				id
				name
				state
				progress
				priority
				lead { displayName }
				teams { nodes { key } }
				initiatives { nodes { name } }
			}
		}
	}`, filterClause)

	var result struct {
		Projects struct {
			Nodes []ProjectFull `json:"nodes"`
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
		if p.Teams != nil {
			for _, t := range p.Teams.Nodes {
				teamKeys = append(teamKeys, t.Key)
			}
		}
		teams := "-"
		if len(teamKeys) > 0 {
			teams = fmt.Sprintf("[%s]", joinStrings(teamKeys, ", "))
		}
		fmt.Printf("%-40s %-12s %3.0f%%  %s\n", p.Name, p.State, p.Progress*100, teams)
	}
	return nil
}

func runProjectsView(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	output, _ := cmd.Root().PersistentFlags().GetString("output")

	projectID, err := resolveProjectID(client, args[0])
	if err != nil {
		return err
	}

	query := `query($id: String!) {
		project(id: $id) {
			id
			name
			description
			state
			progress
			priority
			startDate
			targetDate
			url
			lead { displayName }
			teams { nodes { key name } }
			initiatives { nodes { name } }
			createdAt
			updatedAt
		}
	}`

	var result struct {
		Project ProjectFull `json:"project"`
	}

	if err := client.DoInto(query, map[string]any{"id": projectID}, &result); err != nil {
		return err
	}

	if output == "json" {
		return printJSON(result.Project)
	}

	p := result.Project
	fmt.Printf("%s\n", p.Name)
	fmt.Printf("State:    %s\n", p.State)
	fmt.Printf("Progress: %.0f%%\n", p.Progress*100)
	fmt.Printf("Priority: %s\n", priorityName(p.Priority))
	if p.Lead != nil {
		fmt.Printf("Lead:     %s\n", p.Lead.DisplayName)
	}
	if p.Teams != nil && len(p.Teams.Nodes) > 0 {
		var names []string
		for _, t := range p.Teams.Nodes {
			names = append(names, t.Key)
		}
		fmt.Printf("Teams:    %s\n", strings.Join(names, ", "))
	}
	if p.StartDate != "" {
		fmt.Printf("Start:    %s\n", p.StartDate)
	}
	if p.TargetDate != "" {
		fmt.Printf("Target:   %s\n", p.TargetDate)
	}
	if p.Initiatives != nil && len(p.Initiatives.Nodes) > 0 {
		var names []string
		for _, i := range p.Initiatives.Nodes {
			names = append(names, i.Name)
		}
		fmt.Printf("Initiatives: %s\n", strings.Join(names, ", "))
	}
	if p.Description != "" {
		fmt.Printf("\n%s\n", p.Description)
	}
	fmt.Printf("\n%s\n", p.URL)
	return nil
}

func runProjectsCreate(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	teamKey, _ := cmd.Flags().GetString("team")
	description, _ := cmd.Flags().GetString("description")
	targetDate, _ := cmd.Flags().GetString("target-date")
	startDate, _ := cmd.Flags().GetString("start-date")
	leadMe, _ := cmd.Flags().GetBool("lead-me")
	initiativeName, _ := cmd.Flags().GetString("initiative")
	priority, _ := cmd.Flags().GetInt("priority")
	output, _ := cmd.Root().PersistentFlags().GetString("output")

	teamID, err := resolveTeamID(client, teamKey)
	if err != nil {
		return err
	}

	vars := map[string]any{
		"name":    name,
		"teamIds": []string{teamID},
	}
	if description != "" {
		vars["description"] = description
	}
	if targetDate != "" {
		vars["targetDate"] = targetDate
	}
	if startDate != "" {
		vars["startDate"] = startDate
	}
	if priority > 0 {
		vars["priority"] = priority
	}
	if leadMe {
		meID, err := resolveMe(client)
		if err != nil {
			return err
		}
		vars["leadId"] = meID
	}

	query := `mutation($input: ProjectCreateInput!) {
		projectCreate(input: $input) {
			project {
				id
				name
				state
				url
			}
		}
	}`

	var result struct {
		ProjectCreate struct {
			Project ProjectFull `json:"project"`
		} `json:"projectCreate"`
	}

	if err := client.DoInto(query, map[string]any{"input": vars}, &result); err != nil {
		return err
	}

	p := result.ProjectCreate.Project

	// Link to initiative if specified
	if initiativeName != "" {
		initiativeID, err := resolveInitiativeID(client, initiativeName)
		if err != nil {
			return fmt.Errorf("project created but failed to link initiative: %w", err)
		}
		linkQuery := `mutation($input: InitiativeToProjectCreateInput!) {
			initiativeToProjectCreate(input: $input) {
				initiativeToProject {
					initiative { name }
				}
			}
		}`
		linkInput := map[string]any{
			"initiativeId": initiativeID,
			"projectId":    p.ID,
		}
		var linkResult struct {
			InitiativeToProjectCreate struct {
				InitiativeToProject struct {
					Initiative struct{ Name string } `json:"initiative"`
				} `json:"initiativeToProject"`
			} `json:"initiativeToProjectCreate"`
		}
		if err := client.DoInto(linkQuery, map[string]any{"input": linkInput}, &linkResult); err != nil {
			return fmt.Errorf("project created but failed to link initiative: %w", err)
		}
	}

	if output == "json" {
		return printJSON(p)
	}

	msg := fmt.Sprintf("Created project: %s [%s]\n%s\n", p.Name, p.State, p.URL)
	if initiativeName != "" {
		msg = fmt.Sprintf("Created project: %s [%s] (linked to %q)\n%s\n", p.Name, p.State, initiativeName, p.URL)
	}
	fmt.Print(msg)
	return nil
}

func runProjectsUpdate(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	output, _ := cmd.Root().PersistentFlags().GetString("output")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	targetDate, _ := cmd.Flags().GetString("target-date")
	startDate, _ := cmd.Flags().GetString("start-date")
	leadMe, _ := cmd.Flags().GetBool("lead-me")
	priority, _ := cmd.Flags().GetInt("priority")

	projectID, err := resolveProjectID(client, args[0])
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
	if targetDate != "" {
		vars["targetDate"] = targetDate
	}
	if startDate != "" {
		vars["startDate"] = startDate
	}
	if priority >= 0 {
		vars["priority"] = priority
	}
	if leadMe {
		meID, err := resolveMe(client)
		if err != nil {
			return err
		}
		vars["leadId"] = meID
	}

	if len(vars) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query := `mutation($id: String!, $input: ProjectUpdateInput!) {
		projectUpdate(id: $id, input: $input) {
			project {
				id
				name
				state
				url
			}
		}
	}`

	var result struct {
		ProjectUpdate struct {
			Project ProjectFull `json:"project"`
		} `json:"projectUpdate"`
	}

	if err := client.DoInto(query, map[string]any{"id": projectID, "input": vars}, &result); err != nil {
		return err
	}

	p := result.ProjectUpdate.Project
	if output == "json" {
		return printJSON(p)
	}

	fmt.Printf("Updated project: %s [%s]\n%s\n", p.Name, p.State, p.URL)
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
