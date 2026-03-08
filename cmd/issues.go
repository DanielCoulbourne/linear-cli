package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DanielCoulbourne/orch/tools/linear-cli/internal/api"
	"github.com/spf13/cobra"
)

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Manage Linear issues",
	Long:  `List, view, create, and update issues in Linear.`,
}

var issuesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	Long: `List issues with optional filters.

Examples:
  linear issues list                          # Your assigned issues
  linear issues list --team ENG               # Issues for team ENG
  linear issues list --status "In Progress"   # Filter by status
  linear issues list --project "My Project"   # Filter by project
  linear issues list --limit 50               # Show more results
  linear issues list -o json                  # JSON output`,
	RunE: runIssuesList,
}

var issuesViewCmd = &cobra.Command{
	Use:   "view [ID]",
	Short: "View a single issue",
	Long: `View details of a specific issue by its identifier.

Examples:
  linear issues view ENG-123
  linear issues view ENG-123 -o json`,
	Args: cobra.ExactArgs(1),
	RunE: runIssuesView,
}

var issuesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Long: `Create a new issue in Linear.

Examples:
  linear issues create --title "Fix bug" --team ENG
  linear issues create --title "New feature" --team ENG --description "Details here" --priority 2
  linear issues create --title "Urgent" --team ENG --status "In Progress" --assignee-me
  linear issues create --title "Task" --team ENG --project "My Project"`,
	RunE: runIssuesCreate,
}

var issuesUpdateCmd = &cobra.Command{
	Use:   "update [ID]",
	Short: "Update an existing issue",
	Long: `Update fields on an existing issue.

Examples:
  linear issues update ENG-123 --status "Done"
  linear issues update ENG-123 --title "New title" --priority 1
  linear issues update ENG-123 --assignee-me
  linear issues update ENG-123 --project "My Project"`,
	Args: cobra.ExactArgs(1),
	RunE: runIssuesUpdate,
}

func init() {
	rootCmd.AddCommand(issuesCmd)
	issuesCmd.AddCommand(issuesListCmd)
	issuesCmd.AddCommand(issuesViewCmd)
	issuesCmd.AddCommand(issuesCreateCmd)
	issuesCmd.AddCommand(issuesUpdateCmd)

	// list flags
	issuesListCmd.Flags().String("team", "", "Filter by team key (e.g. ENG)")
	issuesListCmd.Flags().String("status", "", "Filter by status name")
	issuesListCmd.Flags().String("assignee", "", "Filter by assignee (use 'me' for yourself)")
	issuesListCmd.Flags().String("project", "", "Filter by project name")
	issuesListCmd.Flags().Int("limit", 25, "Maximum number of results")

	// create flags
	issuesCreateCmd.Flags().String("title", "", "Issue title (required)")
	issuesCreateCmd.Flags().String("team", "", "Team key (required, e.g. ENG)")
	issuesCreateCmd.Flags().String("description", "", "Issue description (markdown)")
	issuesCreateCmd.Flags().Int("priority", 0, "Priority: 0=none, 1=urgent, 2=high, 3=medium, 4=low")
	issuesCreateCmd.Flags().String("status", "", "Status name")
	issuesCreateCmd.Flags().Bool("assignee-me", false, "Assign to yourself")
	issuesCreateCmd.Flags().String("project", "", "Project name")
	issuesCreateCmd.MarkFlagRequired("title")
	issuesCreateCmd.MarkFlagRequired("team")

	// update flags
	issuesUpdateCmd.Flags().String("title", "", "New title")
	issuesUpdateCmd.Flags().String("status", "", "New status name")
	issuesUpdateCmd.Flags().String("description", "", "New description")
	issuesUpdateCmd.Flags().Int("priority", -1, "New priority")
	issuesUpdateCmd.Flags().Bool("assignee-me", false, "Assign to yourself")
	issuesUpdateCmd.Flags().String("project", "", "Project name")
}

func runIssuesList(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	team, _ := cmd.Flags().GetString("team")
	status, _ := cmd.Flags().GetString("status")
	assignee, _ := cmd.Flags().GetString("assignee")
	projectName, _ := cmd.Flags().GetString("project")
	limit, _ := cmd.Flags().GetInt("limit")
	output, _ := cmd.Root().PersistentFlags().GetString("output")

	var filters []string
	if team != "" {
		filters = append(filters, fmt.Sprintf(`team: { key: { eq: "%s" } }`, team))
	}
	if status != "" {
		filters = append(filters, fmt.Sprintf(`state: { name: { eq: "%s" } }`, status))
	}
	if assignee == "me" {
		filters = append(filters, `assignee: { isMe: { eq: true } }`)
	} else if assignee != "" {
		filters = append(filters, fmt.Sprintf(`assignee: { displayName: { containsIgnoreCase: "%s" } }`, assignee))
	}
	if projectName != "" {
		projectID, err := resolveProjectID(client, projectName)
		if err != nil {
			return err
		}
		filters = append(filters, fmt.Sprintf(`project: { id: { eq: "%s" } }`, projectID))
	}

	filterStr := ""
	if len(filters) > 0 {
		filterStr = fmt.Sprintf("filter: { %s }", strings.Join(filters, ", "))
	}

	query := fmt.Sprintf(`query {
		issues(first: %d, %s, orderBy: updatedAt) {
			nodes {
				identifier
				title
				state { name }
				priority
				assignee { displayName }
				team { key }
				project { name }
				updatedAt
			}
		}
	}`, limit, filterStr)

	var result struct {
		Issues struct {
			Nodes []Issue `json:"nodes"`
		} `json:"issues"`
	}

	if err := client.DoInto(query, nil, &result); err != nil {
		return err
	}

	if output == "json" {
		return printJSON(result.Issues.Nodes)
	}

	if len(result.Issues.Nodes) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	for _, issue := range result.Issues.Nodes {
		assigneeName := "-"
		if issue.Assignee != nil {
			assigneeName = issue.Assignee.DisplayName
		}
		fmt.Printf("%-12s %-14s %-20s %s [%s]\n",
			issue.Identifier, issue.State.Name, assigneeName, issue.Title, issue.Team.Key)
	}
	return nil
}

func runIssuesView(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	output, _ := cmd.Root().PersistentFlags().GetString("output")

	query := `query($id: String!) {
		issue(id: $id) {
			identifier
			title
			description
			state { name }
			priority
			assignee { displayName }
			team { key name }
			project { name }
			cycle { name number }
			labels { nodes { name } }
			createdAt
			updatedAt
			url
		}
	}`

	var result struct {
		Issue Issue `json:"issue"`
	}

	if err := client.DoInto(query, map[string]any{"id": args[0]}, &result); err != nil {
		return err
	}

	if output == "json" {
		return printJSON(result.Issue)
	}

	i := result.Issue
	fmt.Printf("%s: %s\n", i.Identifier, i.Title)
	fmt.Printf("Status:   %s\n", i.State.Name)
	fmt.Printf("Priority: %s\n", priorityName(i.Priority))
	fmt.Printf("Team:     %s\n", i.Team.Key)
	if i.Assignee != nil {
		fmt.Printf("Assignee: %s\n", i.Assignee.DisplayName)
	}
	if i.Project != nil {
		fmt.Printf("Project:  %s\n", i.Project.Name)
	}
	if i.Cycle != nil {
		fmt.Printf("Cycle:    %s (#%d)\n", i.Cycle.Name, i.Cycle.Number)
	}
	if i.Labels != nil && len(i.Labels.Nodes) > 0 {
		var names []string
		for _, l := range i.Labels.Nodes {
			names = append(names, l.Name)
		}
		fmt.Printf("Labels:   %s\n", strings.Join(names, ", "))
	}
	if i.Description != "" {
		fmt.Printf("\n%s\n", i.Description)
	}
	fmt.Printf("\n%s\n", i.URL)
	return nil
}

func runIssuesCreate(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	title, _ := cmd.Flags().GetString("title")
	teamKey, _ := cmd.Flags().GetString("team")
	description, _ := cmd.Flags().GetString("description")
	priority, _ := cmd.Flags().GetInt("priority")
	statusName, _ := cmd.Flags().GetString("status")
	assigneeMe, _ := cmd.Flags().GetBool("assignee-me")
	output, _ := cmd.Root().PersistentFlags().GetString("output")

	teamID, err := resolveTeamID(client, teamKey)
	if err != nil {
		return err
	}

	vars := map[string]any{
		"title":  title,
		"teamId": teamID,
	}
	if description != "" {
		vars["description"] = description
	}
	if priority > 0 {
		vars["priority"] = priority
	}
	if assigneeMe {
		meID, err := resolveMe(client)
		if err != nil {
			return err
		}
		vars["assigneeId"] = meID
	}
	if statusName != "" {
		stateID, err := resolveStateID(client, teamID, statusName)
		if err != nil {
			return err
		}
		vars["stateId"] = stateID
	}
	projectName, _ := cmd.Flags().GetString("project")
	if projectName != "" {
		projectID, err := resolveProjectID(client, projectName)
		if err != nil {
			return err
		}
		vars["projectId"] = projectID
	}

	query := `mutation($input: IssueCreateInput!) {
		issueCreate(input: $input) {
			issue {
				identifier
				title
				state { name }
				url
			}
		}
	}`

	var result struct {
		IssueCreate struct {
			Issue Issue `json:"issue"`
		} `json:"issueCreate"`
	}

	if err := client.DoInto(query, map[string]any{"input": vars}, &result); err != nil {
		return err
	}

	i := result.IssueCreate.Issue
	if output == "json" {
		return printJSON(i)
	}

	fmt.Printf("Created %s: %s [%s]\n%s\n", i.Identifier, i.Title, i.State.Name, i.URL)
	return nil
}

func runIssuesUpdate(cmd *cobra.Command, args []string) error {
	client, err := api.NewClient()
	if err != nil {
		return err
	}

	output, _ := cmd.Root().PersistentFlags().GetString("output")
	title, _ := cmd.Flags().GetString("title")
	statusName, _ := cmd.Flags().GetString("status")
	description, _ := cmd.Flags().GetString("description")
	priority, _ := cmd.Flags().GetInt("priority")
	assigneeMe, _ := cmd.Flags().GetBool("assignee-me")

	issueData, err := fetchIssueByIdentifier(client, args[0])
	if err != nil {
		return err
	}

	vars := map[string]any{}
	if title != "" {
		vars["title"] = title
	}
	if description != "" {
		vars["description"] = description
	}
	if priority >= 0 {
		vars["priority"] = priority
	}
	if assigneeMe {
		meID, err := resolveMe(client)
		if err != nil {
			return err
		}
		vars["assigneeId"] = meID
	}
	if statusName != "" {
		stateID, err := resolveStateID(client, issueData.TeamID, statusName)
		if err != nil {
			return err
		}
		vars["stateId"] = stateID
	}
	projectName, _ := cmd.Flags().GetString("project")
	if projectName != "" {
		projectID, err := resolveProjectID(client, projectName)
		if err != nil {
			return err
		}
		vars["projectId"] = projectID
	}

	if len(vars) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query := `mutation($id: String!, $input: IssueUpdateInput!) {
		issueUpdate(id: $id, input: $input) {
			issue {
				identifier
				title
				state { name }
				url
			}
		}
	}`

	var result struct {
		IssueUpdate struct {
			Issue Issue `json:"issue"`
		} `json:"issueUpdate"`
	}

	if err := client.DoInto(query, map[string]any{"id": issueData.ID, "input": vars}, &result); err != nil {
		return err
	}

	i := result.IssueUpdate.Issue
	if output == "json" {
		return printJSON(i)
	}

	fmt.Printf("Updated %s: %s [%s]\n%s\n", i.Identifier, i.Title, i.State.Name, i.URL)
	return nil
}

type Issue struct {
	ID          string   `json:"id,omitempty"`
	Identifier  string   `json:"identifier"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	State       State    `json:"state"`
	Priority    int      `json:"priority"`
	Assignee    *User    `json:"assignee,omitempty"`
	Team        TeamRef  `json:"team"`
	Project     *Project `json:"project,omitempty"`
	Cycle       *Cycle   `json:"cycle,omitempty"`
	Labels      *Labels  `json:"labels,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	UpdatedAt   string   `json:"updatedAt,omitempty"`
	URL         string   `json:"url,omitempty"`
}

type State struct {
	Name string `json:"name"`
}

type User struct {
	DisplayName string `json:"displayName"`
}

type TeamRef struct {
	Key  string `json:"key"`
	Name string `json:"name,omitempty"`
}

type Project struct {
	Name string `json:"name"`
}

type Cycle struct {
	Name   string `json:"name"`
	Number int    `json:"number"`
}

type Labels struct {
	Nodes []Label `json:"nodes"`
}

type Label struct {
	Name string `json:"name"`
}

type issueIDData struct {
	ID     string
	TeamID string
}

func fetchIssueByIdentifier(client *api.Client, identifier string) (*issueIDData, error) {
	query := `query($id: String!) {
		issue(id: $id) {
			id
			team { id }
		}
	}`
	var result struct {
		Issue struct {
			ID   string `json:"id"`
			Team struct {
				ID string `json:"id"`
			} `json:"team"`
		} `json:"issue"`
	}
	if err := client.DoInto(query, map[string]any{"id": identifier}, &result); err != nil {
		return nil, err
	}
	return &issueIDData{ID: result.Issue.ID, TeamID: result.Issue.Team.ID}, nil
}

func resolveTeamID(client *api.Client, key string) (string, error) {
	query := `query {
		teams {
			nodes { id key }
		}
	}`
	var result struct {
		Teams struct {
			Nodes []struct {
				ID  string `json:"id"`
				Key string `json:"key"`
			} `json:"nodes"`
		} `json:"teams"`
	}
	if err := client.DoInto(query, nil, &result); err != nil {
		return "", err
	}
	for _, t := range result.Teams.Nodes {
		if strings.EqualFold(t.Key, key) {
			return t.ID, nil
		}
	}
	return "", fmt.Errorf("team %q not found", key)
}

func resolveMe(client *api.Client) (string, error) {
	query := `query { viewer { id } }`
	var result struct {
		Viewer struct {
			ID string `json:"id"`
		} `json:"viewer"`
	}
	if err := client.DoInto(query, nil, &result); err != nil {
		return "", err
	}
	return result.Viewer.ID, nil
}

func resolveStateID(client *api.Client, teamID, stateName string) (string, error) {
	query := `query($teamId: ID!) {
		workflowStates(filter: { team: { id: { eq: $teamId } } }) {
			nodes { id name }
		}
	}`
	var result struct {
		WorkflowStates struct {
			Nodes []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"nodes"`
		} `json:"workflowStates"`
	}
	if err := client.DoInto(query, map[string]any{"teamId": teamID}, &result); err != nil {
		return "", err
	}
	for _, s := range result.WorkflowStates.Nodes {
		if strings.EqualFold(s.Name, stateName) {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("state %q not found for team", stateName)
}

func priorityName(p int) string {
	switch p {
	case 1:
		return "Urgent"
	case 2:
		return "High"
	case 3:
		return "Medium"
	case 4:
		return "Low"
	default:
		return "None"
	}
}

func resolveProjectID(client *api.Client, name string) (string, error) {
	query := fmt.Sprintf(`query {
		projects(filter: { name: { containsIgnoreCase: "%s" } }) {
			nodes { id name }
		}
	}`, name)
	var result struct {
		Projects struct {
			Nodes []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"nodes"`
		} `json:"projects"`
	}
	if err := client.DoInto(query, nil, &result); err != nil {
		return "", err
	}
	if len(result.Projects.Nodes) == 0 {
		return "", fmt.Errorf("project %q not found", name)
	}
	// Prefer exact (case-insensitive) match
	for _, p := range result.Projects.Nodes {
		if strings.EqualFold(p.Name, name) {
			return p.ID, nil
		}
	}
	// Fall back to first partial match
	return result.Projects.Nodes[0].ID, nil
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
