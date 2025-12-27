package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
	"github.com/salmonumbrella/notion-cli/internal/skill"
)

type ResolveCandidate struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	Title  string `json:"title,omitempty"`
	URL    string `json:"url,omitempty"`

	Source string `json:"source,omitempty"` // "skill" or "search"
	Alias  string `json:"alias,omitempty"`
	Exact  bool   `json:"exact,omitempty"`
}

type ResolveResponse struct {
	Object     string             `json:"object"`
	Query      string             `json:"query"`
	Filter     string             `json:"filter,omitempty"`
	Results    []ResolveCandidate `json:"results"`
	HasMore    bool               `json:"has_more"`
	NextCursor *string            `json:"next_cursor,omitempty"`
	Meta       map[string]any     `json:"_meta,omitempty"`
}

func newResolveCmd() *cobra.Command {
	var filterType string
	var pageSize int
	var startCursor string
	var all bool
	var exactOnly bool

	cmd := &cobra.Command{
		Use:     "resolve <query>",
		Aliases: []string{"res", "r"},
		Short:   "Resolve a name/alias to Notion IDs (agent-friendly)",
		Long: `Resolve a query to candidate Notion objects.

This command is designed for agents: it returns a compact list of candidates
with IDs and titles so follow-up actions can use IDs directly.

Resolution sources:
  1. Skill file aliases (~/.claude/skills/notion-cli/notion-cli.md)
  2. Notion search API (pages + databases)

Use global --results-only to output just the candidates array.

Examples:
  ntn resolve "Meeting Notes"
  ntn resolve "Projects" --type database
  ntn resolve standup        # skill alias
  ntn resolve "Meeting Notes" --exact`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			query := strings.TrimSpace(args[0])
			if query == "" {
				return fmt.Errorf("query is required")
			}

			// Validate filter type.
			filterType = strings.ToLower(strings.TrimSpace(filterType))
			switch filterType {
			case "", "any":
				filterType = ""
			case "page", "database", "user":
				// ok
			default:
				return errors.NewUserError(
					fmt.Sprintf("invalid --type %q", filterType),
					"Use one of: any, page, database, user",
				)
			}

			limit := output.LimitFromContext(ctx)
			pageSize = capPageSize(pageSize, limit)
			if pageSize > NotionMaxPageSize {
				return fmt.Errorf("page-size must be between 1 and %d", NotionMaxPageSize)
			}
			if pageSize == 0 {
				pageSize = 10
			}

			sf := SkillFileFromContext(ctx)
			if sf == nil {
				sf = &skill.SkillFile{}
			}

			// First: skill file matches (fast, no network).
			candidates := resolveCandidatesFromSkill(sf, query, filterType)

			// Then: Notion search.
			client, err := clientFromContext(ctx)
			if err != nil {
				return err
			}

			filter := buildSearchFilter(filterType)

			var results []map[string]interface{}
			var nextCursor *string
			var hasMore bool

			if all {
				allResults, nc, hm, err := fetchAllPages(ctx, startCursor, pageSize, limit, func(ctx context.Context, cursor string, pageSize int) ([]map[string]interface{}, *string, bool, error) {
					req := &notion.SearchRequest{
						Query:       query,
						Filter:      filter,
						StartCursor: cursor,
						PageSize:    pageSize,
					}
					res, err := client.Search(ctx, req)
					if err != nil {
						return nil, nil, false, err
					}
					return res.Results, res.NextCursor, res.HasMore, nil
				})
				if err != nil {
					return fmt.Errorf("failed to search: %w", err)
				}
				results = allResults
				nextCursor = nc
				hasMore = hm
			} else {
				req := &notion.SearchRequest{
					Query:       query,
					Filter:      filter,
					StartCursor: startCursor,
					PageSize:    pageSize,
				}
				res, err := client.Search(ctx, req)
				if err != nil {
					return fmt.Errorf("failed to search: %w", err)
				}
				results = res.Results
				nextCursor = res.NextCursor
				hasMore = res.HasMore
			}

			// Add search candidates (dedup against skill candidates).
			seen := make(map[string]bool, len(candidates))
			for _, c := range candidates {
				seen[c.Object+"\x00"+c.ID] = true
			}
			for _, r := range results {
				id, _ := r["id"].(string)
				if id == "" {
					continue
				}
				obj, _ := r["object"].(string)
				if obj == "data_source" {
					obj = "database"
				}
				title := extractResultTitle(r)
				url, _ := r["url"].(string)
				exact := strings.EqualFold(strings.TrimSpace(title), query) && title != ""
				key := obj + "\x00" + id
				if seen[key] {
					continue
				}
				seen[key] = true
				candidates = append(candidates, ResolveCandidate{
					ID:     id,
					Object: obj,
					Title:  title,
					URL:    url,
					Source: "search",
					Exact:  exact,
				})
			}

			if exactOnly {
				filtered := make([]ResolveCandidate, 0, len(candidates))
				for _, c := range candidates {
					if c.Exact {
						filtered = append(filtered, c)
					}
				}
				candidates = filtered
			}

			resp := ResolveResponse{
				Object:     "resolve_list",
				Query:      query,
				Filter:     filterType,
				Results:    candidates,
				HasMore:    hasMore,
				NextCursor: nextCursor,
				Meta: map[string]any{
					"skill_matches": countSource(candidates, "skill"),
				},
			}

			printer := printerForContext(ctx)
			return printer.Print(ctx, resp)
		},
	}

	cmd.Flags().StringVar(&filterType, "type", "any", "Filter candidates by type (any|page|database|user)")
	cmd.Flags().IntVar(&pageSize, "page-size", 10, "Number of results per page (max 100)")
	cmd.Flags().StringVar(&startCursor, "start-cursor", "", "Pagination cursor")
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages of results (may be slow for large workspaces)")
	cmd.Flags().BoolVar(&exactOnly, "exact", false, "Only return exact title matches (case-insensitive) and exact skill alias matches")

	return cmd
}

func resolveCandidatesFromSkill(sf *skill.SkillFile, query, filterType string) []ResolveCandidate {
	if sf == nil {
		return nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	var out []ResolveCandidate

	if db, ok := sf.Databases[query]; ok {
		out = append(out, ResolveCandidate{
			ID:     db.ID,
			Object: "database",
			Title:  db.Name,
			Source: "skill",
			Alias:  db.Alias,
			Exact:  true,
		})
	}
	if u, ok := sf.Users[query]; ok {
		out = append(out, ResolveCandidate{
			ID:     u.ID,
			Object: "user",
			Title:  u.Name,
			Source: "skill",
			Alias:  u.Alias,
			Exact:  true,
		})
	}
	if a, ok := sf.Aliases[query]; ok {
		obj := a.Type
		if obj == "data_source" {
			obj = "database"
		}
		out = append(out, ResolveCandidate{
			ID:     a.TargetID,
			Object: obj,
			Source: "skill",
			Alias:  a.Alias,
			Exact:  true,
		})
	}

	if filterType != "" {
		filtered := out[:0]
		for _, c := range out {
			if c.Object == filterType {
				filtered = append(filtered, c)
			}
		}
		out = filtered
	}

	return out
}

func countSource(items []ResolveCandidate, source string) int {
	n := 0
	for _, it := range items {
		if it.Source == source {
			n++
		}
	}
	return n
}
