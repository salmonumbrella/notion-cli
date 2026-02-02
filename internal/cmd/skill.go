// Package cmd contains the CLI commands for notion-cli.
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/skill"
)

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage the notion-cli skill file for agents",
		Long: `The skill file provides aliases and context for AI agents using the CLI.

Run 'notion skill init' after authentication to generate a skill file
tailored to your workspace.`,
	}

	cmd.AddCommand(newSkillInitCmd())
	cmd.AddCommand(newSkillSyncCmd())
	cmd.AddCommand(newSkillPathCmd())
	cmd.AddCommand(newSkillEditCmd())

	return cmd
}

func newSkillInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize skill file by scanning your workspace",
		Long: `Scans your Notion workspace and guides you through creating a skill file.

The wizard will:
1. Discover all databases and datasources
2. Ask you to configure aliases for each
3. Discover all users
4. Set up user aliases (including "me" for yourself)
5. Allow custom aliases for frequently-used pages

The generated skill file is saved to ~/.claude/skills/notion-cli/notion-cli.md`,
		RunE: runSkillInit,
	}
}

func newSkillSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Update skill file with current workspace state",
		Long:  `Re-scans your workspace and updates the skill file, preserving existing aliases.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement sync
			return fmt.Errorf("not implemented yet")
		},
	}
}

func newSkillPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the skill file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(skill.DefaultPath())
			return nil
		},
	}
}

func newSkillEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open skill file in your editor",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement edit (open in $EDITOR)
			return fmt.Errorf("not implemented yet")
		},
	}
}

// workspaceData holds the scanned workspace information
type workspaceData struct {
	Databases   []notion.Database
	DataSources []*notion.DataSource
	Users       []*notion.User
	CurrentUser *notion.User
}

func runSkillInit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	stderr := stderrFromContext(ctx)

	// Get API client
	token, err := GetTokenFromContext(ctx)
	if err != nil {
		return errors.AuthRequiredError(err)
	}

	client := NewNotionClient(ctx, token)

	_, _ = fmt.Fprintln(stderr, "Scanning your Notion workspace...")
	_, _ = fmt.Fprintln(stderr, "")

	// Scan workspace
	data, err := scanWorkspace(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to scan workspace: %w", err)
	}

	_, _ = fmt.Fprintf(stderr, "Found:\n")
	_, _ = fmt.Fprintf(stderr, "  - %d databases\n", len(data.Databases))
	_, _ = fmt.Fprintf(stderr, "  - %d datasources\n", len(data.DataSources))
	_, _ = fmt.Fprintf(stderr, "  - %d users\n", len(data.Users))
	_, _ = fmt.Fprintln(stderr, "")

	// Create empty skill file (configuration will come in next tasks)
	skillFile := &skill.SkillFile{
		Databases: make(map[string]skill.DatabaseAlias),
		Users:     make(map[string]skill.UserAlias),
		Aliases:   make(map[string]skill.CustomAlias),
	}

	// TODO: Interactive configuration (Tasks 5-7)

	// For now, just save empty skill file to verify the flow works
	if err := skillFile.Save(); err != nil {
		return fmt.Errorf("failed to save skill file: %w", err)
	}

	_, _ = fmt.Fprintf(stderr, "Skill file created: %s\n", skill.DefaultPath())
	return nil
}

func scanWorkspace(ctx context.Context, client *notion.Client) (*workspaceData, error) {
	data := &workspaceData{}

	// Get current user (bot user)
	me, err := client.GetSelf(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	data.CurrentUser = me

	// Get all users with pagination
	var allUsers []*notion.User
	var userCursor string
	for {
		opts := &notion.ListUsersOptions{
			StartCursor: userCursor,
			PageSize:    100,
		}
		users, err := client.ListUsers(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}
		allUsers = append(allUsers, users.Results...)
		if !users.HasMore || users.NextCursor == nil {
			break
		}
		userCursor = *users.NextCursor
	}
	data.Users = allUsers

	// Search for data sources with pagination (API 2025-09-03+ uses data_source instead of database)
	// Track unique databases by ID (to avoid duplicates from multiple data sources)
	seenDBs := make(map[string]bool)
	var searchCursor string

	for {
		searchResp, err := client.Search(ctx, &notion.SearchRequest{
			Filter: map[string]interface{}{
				"property": "object",
				"value":    "data_source",
			},
			StartCursor: searchCursor,
			PageSize:    100,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search databases: %w", err)
		}

		// Convert search results to DataSource structs and extract parent database info
		for _, result := range searchResp.Results {
			if result["object"] == "data_source" {
				ds := parseDataSourceResult(result)
				data.DataSources = append(data.DataSources, ds)

				// Extract parent database info if available
				if parent, ok := result["parent"].(map[string]interface{}); ok {
					if dbID, ok := parent["database_id"].(string); ok && !seenDBs[dbID] {
						seenDBs[dbID] = true
						db := buildDatabaseFromDataSource(dbID, ds, result)
						data.Databases = append(data.Databases, db)
					}
				}
			}
		}

		if !searchResp.HasMore || searchResp.NextCursor == nil {
			break
		}
		searchCursor = *searchResp.NextCursor
	}

	return data, nil
}

// parseDataSourceResult converts a search result map to a DataSource struct
func parseDataSourceResult(result map[string]interface{}) *notion.DataSource {
	ds := &notion.DataSource{
		Object: "data_source",
	}
	if id, ok := result["id"].(string); ok {
		ds.ID = id
	}
	if title, ok := result["title"].([]interface{}); ok {
		ds.Title = make([]notion.RichText, 0, len(title))
		for _, t := range title {
			if m, ok := t.(map[string]interface{}); ok {
				rt := notion.RichText{}
				if typ, ok := m["type"].(string); ok {
					rt.Type = typ
				}
				if plainText, ok := m["plain_text"].(string); ok {
					rt.PlainText = plainText
				}
				ds.Title = append(ds.Title, rt)
			}
		}
	}
	if props, ok := result["properties"].(map[string]interface{}); ok {
		ds.Properties = props
	}
	if parent, ok := result["parent"].(map[string]interface{}); ok {
		ds.Parent = parent
	}
	return ds
}

// buildDatabaseFromDataSource creates a Database struct from data source info
func buildDatabaseFromDataSource(dbID string, ds *notion.DataSource, result map[string]interface{}) notion.Database {
	db := notion.Database{
		Object: "database",
		ID:     dbID,
	}
	// Try to get database title from the data source title
	if len(ds.Title) > 0 {
		db.Title = []map[string]interface{}{
			{
				"type":       "text",
				"plain_text": ds.Title[0].PlainText,
				"text": map[string]interface{}{
					"content": ds.Title[0].PlainText,
				},
			},
		}
	}
	// Copy properties from data source
	if props, ok := result["properties"].(map[string]interface{}); ok {
		db.Properties = make(map[string]map[string]interface{})
		for k, v := range props {
			if propMap, ok := v.(map[string]interface{}); ok {
				db.Properties[k] = propMap
			}
		}
	}
	db.DataSources = []notion.DataSourceRef{{ID: ds.ID}}
	return db
}
