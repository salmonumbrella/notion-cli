// Package cmd contains the CLI commands for notion-cli.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/skill"
)

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "skill",
		Aliases: []string{"sk"},
		Short:   "Manage the notion-cli skill file for agents",
		Long: `The skill file provides aliases and context for AI agents using the CLI.

Run 'ntn skill init' after authentication to generate a skill file
tailored to your workspace.`,
	}

	cmd.AddCommand(newSkillInitCmd())
	cmd.AddCommand(newSkillSyncCmd())
	cmd.AddCommand(newSkillPathCmd())
	cmd.AddCommand(newSkillEditCmd())

	return cmd
}

func newSkillInitCmd() *cobra.Command {
	skillPath := skill.DefaultPath()

	return &cobra.Command{
		Use:   "init",
		Short: "Initialize skill file by scanning your workspace",
		Long: fmt.Sprintf(`Scans your Notion workspace and guides you through creating a skill file.

The wizard will:
1. Discover all databases and datasources
2. Ask you to configure aliases for each
3. Discover all users
4. Set up user aliases (including "me" for yourself)
5. Allow custom aliases for frequently-used pages

The generated skill file is saved to %s`, skillPath),
		RunE: runSkillInit,
	}
}

func newSkillSyncCmd() *cobra.Command {
	var addNew bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Update skill file with current workspace state",
		Long:  `Re-scans your workspace and updates the skill file, preserving existing aliases.`,
		RunE:  func(cmd *cobra.Command, args []string) error { return runSkillSync(cmd, addNew) },
	}

	cmd.Flags().BoolVar(&addNew, "add-new", false, "Add newly discovered databases/users with generated aliases (non-interactive)")
	return cmd
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
			return runSkillEdit(cmd.Context())
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
	client, err := clientFromContext(ctx)
	if err != nil {
		return err
	}

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

	// Interactive configuration
	reader := bufio.NewReader(os.Stdin)

	// Configure databases
	if err := configureDatabases(stderr, reader, data, skillFile); err != nil {
		return fmt.Errorf("database configuration failed: %w", err)
	}

	// Configure users
	if err := configureUsers(stderr, reader, data, skillFile); err != nil {
		return fmt.Errorf("user configuration failed: %w", err)
	}

	// Configure custom aliases
	if err := configureCustomAliases(ctx, stderr, reader, client, skillFile); err != nil {
		return fmt.Errorf("custom alias configuration failed: %w", err)
	}

	// Save skill file
	if err := skillFile.Save(); err != nil {
		return fmt.Errorf("failed to save skill file: %w", err)
	}

	_, _ = fmt.Fprintln(stderr, "")
	_, _ = fmt.Fprintf(stderr, "Skill file created: %s\n", skill.DefaultPath())
	_, _ = fmt.Fprintln(stderr, "")
	_, _ = fmt.Fprintln(stderr, "Summary:")
	_, _ = fmt.Fprintf(stderr, "  - %d database aliases\n", len(skillFile.Databases))
	_, _ = fmt.Fprintf(stderr, "  - %d user aliases\n", len(skillFile.Users))
	_, _ = fmt.Fprintf(stderr, "  - %d custom aliases\n", len(skillFile.Aliases))
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

func runSkillSync(cmd *cobra.Command, addNew bool) error {
	ctx := cmd.Context()
	stderr := stderrFromContext(ctx)

	// Require existing file for sync (init is the wizard).
	path := skill.DefaultPath()
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return errors.NewUserError(
				"skill file not found",
				"Run 'ntn skill init' first to create it, then re-run 'ntn skill sync'.",
			)
		}
		return fmt.Errorf("failed to stat skill file: %w", err)
	}

	// Load existing skill file.
	sf, err := skill.Load()
	if err != nil {
		return fmt.Errorf("failed to load skill file: %w", err)
	}

	client, err := clientFromContext(ctx)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(stderr, "Scanning your Notion workspace...")
	data, err := scanWorkspace(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to scan workspace: %w", err)
	}

	// Index discovered entities by ID for quick lookup.
	dbByID := make(map[string]string, len(data.Databases))
	for _, db := range data.Databases {
		name := extractDatabaseTitle(db)
		if name == "" {
			name = "Untitled"
		}
		dbByID[db.ID] = name
	}
	userByID := make(map[string]string, len(data.Users))
	for _, u := range data.Users {
		if u == nil || u.ID == "" {
			continue
		}
		if u.Name == "" {
			continue
		}
		userByID[u.ID] = u.Name
	}

	updatedDBs := 0
	for alias, db := range sf.Databases {
		if name, ok := dbByID[db.ID]; ok && name != "" && name != db.Name {
			db.Name = name
			sf.Databases[alias] = db
			updatedDBs++
		}
	}
	updatedUsers := 0
	for alias, u := range sf.Users {
		if name, ok := userByID[u.ID]; ok && name != "" && name != u.Name {
			u.Name = name
			sf.Users[alias] = u
			updatedUsers++
		}
	}

	addedDBs := 0
	addedUsers := 0
	if addNew {
		seenDBIDs := make(map[string]bool, len(sf.Databases))
		for _, db := range sf.Databases {
			seenDBIDs[db.ID] = true
		}
		seenUserIDs := make(map[string]bool, len(sf.Users))
		for _, u := range sf.Users {
			seenUserIDs[u.ID] = true
		}

		aliasTaken := func(a string) bool {
			if a == "" {
				return true
			}
			if _, ok := sf.Databases[a]; ok {
				return true
			}
			if _, ok := sf.Users[a]; ok {
				return true
			}
			if _, ok := sf.Aliases[a]; ok {
				return true
			}
			return false
		}
		uniqueAlias := func(base string) string {
			base = strings.TrimSpace(base)
			if base == "" {
				base = "alias"
			}
			if !aliasTaken(base) {
				return base
			}
			for i := 2; ; i++ {
				cand := fmt.Sprintf("%s%d", base, i)
				if !aliasTaken(cand) {
					return cand
				}
			}
		}

		for _, db := range data.Databases {
			if db.ID == "" || seenDBIDs[db.ID] {
				continue
			}
			name := extractDatabaseTitle(db)
			if name == "" {
				name = "Untitled"
			}
			alias := uniqueAlias(suggestAlias(name))
			sf.Databases[alias] = skill.DatabaseAlias{
				Alias: alias,
				Name:  name,
				ID:    db.ID,
				// Leave TitleProperty/DefaultStatus empty; init wizard is where
				// those preferences are chosen. Agents can still use aliases.
			}
			addedDBs++
		}

		for _, u := range data.Users {
			if u == nil || u.ID == "" || seenUserIDs[u.ID] {
				continue
			}
			if u.Type == "bot" {
				continue
			}
			name := strings.TrimSpace(u.Name)
			if name == "" {
				continue
			}
			alias := uniqueAlias(suggestUserAlias(name))
			// Never override reserved "me".
			if alias == "me" {
				alias = uniqueAlias("me2")
			}
			sf.Users[alias] = skill.UserAlias{
				Alias: alias,
				Name:  name,
				ID:    u.ID,
			}
			addedUsers++
		}
	}

	if err := sf.Save(); err != nil {
		return fmt.Errorf("failed to save skill file: %w", err)
	}

	_, _ = fmt.Fprintln(stderr, "")
	_, _ = fmt.Fprintf(stderr, "Skill file updated: %s\n", path)
	_, _ = fmt.Fprintf(stderr, "  - %d database names refreshed\n", updatedDBs)
	_, _ = fmt.Fprintf(stderr, "  - %d user names refreshed\n", updatedUsers)
	if addNew {
		_, _ = fmt.Fprintf(stderr, "  - %d databases added\n", addedDBs)
		_, _ = fmt.Fprintf(stderr, "  - %d users added\n", addedUsers)
	} else {
		_, _ = fmt.Fprintln(stderr, "  - new databases/users not added (use --add-new)")
	}
	return nil
}

func runSkillEdit(ctx context.Context) error {
	path := skill.DefaultPath()
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return errors.NewUserError(
				"skill file not found",
				"Run 'ntn skill init' first to create it, then re-run 'ntn skill edit'.",
			)
		}
		return fmt.Errorf("failed to stat skill file: %w", err)
	}

	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("VISUAL"))
	}
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vi"
		}
	}

	// If $EDITOR contains args (e.g. "code -w"), split conservatively.
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return fmt.Errorf("invalid $EDITOR %q", editor)
	}
	bin := parts[0]
	args := append(parts[1:], path)

	c := exec.CommandContext(ctx, bin, args...)
	// $EDITOR requires a real terminal for interactive editing.
	// Context IO streams cannot be used here.
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	// Ensure parent dirs exist (helpful if a user manually deleted them).
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	return c.Run()
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

// configureDatabases prompts user to configure aliases for each database
func configureDatabases(w io.Writer, reader *bufio.Reader, data *workspaceData, sf *skill.SkillFile) error {
	if len(data.Databases) == 0 {
		_, _ = fmt.Fprintln(w, "No databases found in workspace.")
		return nil
	}

	_, _ = fmt.Fprintln(w, "Configure database aliases:")
	_, _ = fmt.Fprintln(w, "(Press Enter to accept suggested values, or type a new value)")
	_, _ = fmt.Fprintln(w, "")

	for _, db := range data.Databases {
		dbName := extractDatabaseTitle(db)
		if dbName == "" {
			dbName = "Untitled"
		}

		_, _ = fmt.Fprintf(w, "Database: %s\n", dbName)

		// Suggest alias
		suggested := suggestAlias(dbName)
		alias := promptWithDefault(w, reader, "  Alias", suggested)
		if alias == "" {
			_, _ = fmt.Fprintln(w, "  Skipping (no alias provided)")
			_, _ = fmt.Fprintln(w, "")
			continue
		}

		// Find title property
		titleProp := findTitleProperty(db)
		titleProperty := promptWithDefault(w, reader, "  Title property", titleProp)

		// Default status
		defaultStatus := promptWithDefault(w, reader, "  Default status", "Todo")

		sf.Databases[alias] = skill.DatabaseAlias{
			Alias:         alias,
			Name:          dbName,
			ID:            db.ID,
			TitleProperty: titleProperty,
			DefaultStatus: defaultStatus,
		}

		_, _ = fmt.Fprintln(w, "")
	}

	return nil
}

// configureUsers prompts user to configure aliases for users
func configureUsers(w io.Writer, reader *bufio.Reader, data *workspaceData, sf *skill.SkillFile) error {
	if len(data.Users) == 0 {
		_, _ = fmt.Fprintln(w, "No users found in workspace.")
		return nil
	}

	_, _ = fmt.Fprintln(w, "Configure user aliases:")
	_, _ = fmt.Fprintln(w, "")

	// First, set up "me" for the current user
	// For OAuth, CurrentUser is the authenticated user
	// For bot tokens, we need to find a human user
	var meUser *notion.User
	if data.CurrentUser != nil && data.CurrentUser.Type == "person" {
		meUser = data.CurrentUser
	} else {
		// Find first person user if current user is a bot
		for _, u := range data.Users {
			if u.Type == "person" {
				meUser = u
				break
			}
		}
	}

	if meUser != nil {
		_, _ = fmt.Fprintf(w, "Setting up 'me' alias for: %s\n", meUser.Name)
		sf.Users["me"] = skill.UserAlias{
			Alias: "me",
			Name:  meUser.Name,
			ID:    meUser.ID,
		}
		_, _ = fmt.Fprintln(w, "")
	}

	// Ask if user wants to set up aliases for other users
	_, _ = fmt.Fprint(w, "Would you like to set up aliases for other users? [y/N]: ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		_, _ = fmt.Fprintln(w, "")
		return nil
	}

	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "(Press Enter to skip a user, or type an alias)")
	_, _ = fmt.Fprintln(w, "")

	for _, user := range data.Users {
		// Skip bots and the "me" user
		if user.Type == "bot" {
			continue
		}
		if meUser != nil && user.ID == meUser.ID {
			continue
		}

		suggested := suggestUserAlias(user.Name)
		_, _ = fmt.Fprintf(w, "User: %s\n", user.Name)
		alias := promptWithDefault(w, reader, "  Alias", suggested)

		if alias == "" || alias == "me" {
			if alias == "me" {
				_, _ = fmt.Fprintln(w, "  Skipping ('me' is reserved)")
			} else {
				_, _ = fmt.Fprintln(w, "  Skipping")
			}
			_, _ = fmt.Fprintln(w, "")
			continue
		}

		sf.Users[alias] = skill.UserAlias{
			Alias: alias,
			Name:  user.Name,
			ID:    user.ID,
		}
		_, _ = fmt.Fprintln(w, "")
	}

	return nil
}

// configureCustomAliases prompts user to create custom aliases for pages/databases
func configureCustomAliases(ctx context.Context, w io.Writer, reader *bufio.Reader, client *notion.Client, sf *skill.SkillFile) error {
	_, _ = fmt.Fprintln(w, "Configure custom aliases:")
	_, _ = fmt.Fprintln(w, "Create shortcuts for frequently-used pages or databases.")
	_, _ = fmt.Fprintln(w, "Enter aliases in the format: 'X for Y' (e.g., 'standup for daily standup notes')")
	_, _ = fmt.Fprintln(w, "Press Enter with no input when done.")
	_, _ = fmt.Fprintln(w, "")

	for {
		_, _ = fmt.Fprint(w, "Alias: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input = strings.TrimSpace(input)

		if input == "" {
			break
		}

		// Parse "X for Y" format
		alias, searchQuery := parseAliasInput(input)
		if alias == "" || searchQuery == "" {
			_, _ = fmt.Fprintln(w, "  Invalid format. Use 'alias for search query' (e.g., 'standup for daily standup')")
			_, _ = fmt.Fprintln(w, "")
			continue
		}

		// Search workspace
		results, err := client.Search(ctx, &notion.SearchRequest{
			Query:    searchQuery,
			PageSize: 10,
		})
		if err != nil {
			_, _ = fmt.Fprintf(w, "  Search failed: %v\n", err)
			_, _ = fmt.Fprintln(w, "")
			continue
		}

		if len(results.Results) == 0 {
			_, _ = fmt.Fprintf(w, "  No results found for '%s'\n", searchQuery)
			_, _ = fmt.Fprintln(w, "")
			continue
		}

		// Display results for selection
		_, _ = fmt.Fprintf(w, "  Found %d results for '%s':\n", len(results.Results), searchQuery)
		for i, result := range results.Results {
			objType, _ := result["object"].(string)
			title := extractSearchResultTitle(result)
			_, _ = fmt.Fprintf(w, "    %d. [%s] %s\n", i+1, objType, title)
		}

		// Let user select
		_, _ = fmt.Fprint(w, "  Select (1-", len(results.Results), ") or 0 to skip: ")
		selInput, _ := reader.ReadString('\n')
		selInput = strings.TrimSpace(selInput)

		selection := 0
		_, _ = fmt.Sscanf(selInput, "%d", &selection)

		if selection < 1 || selection > len(results.Results) {
			_, _ = fmt.Fprintln(w, "  Skipped")
			_, _ = fmt.Fprintln(w, "")
			continue
		}

		selected := results.Results[selection-1]
		objType, _ := selected["object"].(string)
		objID, _ := selected["id"].(string)

		sf.Aliases[alias] = skill.CustomAlias{
			Alias:    alias,
			Type:     objType,
			TargetID: objID,
		}

		_, _ = fmt.Fprintf(w, "  Created alias '%s' -> %s (%s)\n", alias, extractSearchResultTitle(selected), objID)
		_, _ = fmt.Fprintln(w, "")
	}

	return nil
}

// Helper functions

// suggestAlias suggests an alias based on the database name
func suggestAlias(name string) string {
	lower := strings.ToLower(name)
	patterns := map[string]string{
		"issue":    "issues",
		"task":     "tasks",
		"project":  "projects",
		"meeting":  "meetings",
		"calendar": "calendar",
		"content":  "content",
		"sprint":   "sprints",
		"roadmap":  "roadmap",
		"note":     "notes",
		"doc":      "docs",
		"bug":      "bugs",
		"feature":  "features",
		"backlog":  "backlog",
		"ticket":   "tickets",
	}
	for pattern, suggestion := range patterns {
		if strings.Contains(lower, pattern) {
			return suggestion
		}
	}
	// Default: lowercase first word, pluralize if single word
	parts := strings.Fields(lower)
	if len(parts) > 0 {
		word := parts[0]
		// Simple pluralization
		if !strings.HasSuffix(word, "s") {
			word += "s"
		}
		return word
	}
	return ""
}

// suggestUserAlias suggests an alias for a user (lowercase first name)
func suggestUserAlias(name string) string {
	parts := strings.Fields(name)
	if len(parts) > 0 {
		return strings.ToLower(parts[0])
	}
	return ""
}

// extractDatabaseTitle extracts the title from a database
func extractDatabaseTitle(db notion.Database) string {
	if len(db.Title) > 0 {
		if plainText, ok := db.Title[0]["plain_text"].(string); ok {
			return plainText
		}
	}
	return ""
}

// findTitleProperty finds the title property name in a database
func findTitleProperty(db notion.Database) string {
	for propName, propDef := range db.Properties {
		if propType, ok := propDef["type"].(string); ok && propType == "title" {
			return propName
		}
	}
	return "Title"
}

// extractSearchResultTitle extracts the title from a search result
func extractSearchResultTitle(result map[string]interface{}) string {
	// For pages
	if props, ok := result["properties"].(map[string]interface{}); ok {
		for _, prop := range props {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if propType, ok := propMap["type"].(string); ok && propType == "title" {
					if titleArr, ok := propMap["title"].([]interface{}); ok {
						var title strings.Builder
						for _, t := range titleArr {
							if tMap, ok := t.(map[string]interface{}); ok {
								if pt, ok := tMap["plain_text"].(string); ok {
									title.WriteString(pt)
								}
							}
						}
						if title.Len() > 0 {
							return title.String()
						}
					}
				}
			}
		}
	}

	// For databases/data_sources - check title array directly
	if titleArr, ok := result["title"].([]interface{}); ok {
		var title strings.Builder
		for _, t := range titleArr {
			if tMap, ok := t.(map[string]interface{}); ok {
				if pt, ok := tMap["plain_text"].(string); ok {
					title.WriteString(pt)
				}
			}
		}
		if title.Len() > 0 {
			return title.String()
		}
	}

	return "Untitled"
}

// parseAliasInput parses "X for Y" format into alias and search query
func parseAliasInput(input string) (alias, query string) {
	parts := strings.SplitN(input, " for ", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

// promptWithDefault prompts for input with a default value
func promptWithDefault(w io.Writer, reader *bufio.Reader, prompt, defaultValue string) string {
	if defaultValue != "" {
		_, _ = fmt.Fprintf(w, "%s [%s]: ", prompt, defaultValue)
	} else {
		_, _ = fmt.Fprintf(w, "%s: ", prompt)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}
