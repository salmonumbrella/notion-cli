package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/skill"
)

type dataSourceGetter interface {
	GetDataSource(ctx context.Context, dataSourceID string) (*notion.DataSource, error)
}

var uuidLike = regexp.MustCompile(`(?i)^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$`)

func buildDBQueryShorthandFilters(
	ctx context.Context,
	client dataSourceGetter,
	sf *skill.SkillFile,
	dataSourceID string,
	statusProp string,
	statusEquals string,
	assigneeProp string,
	assigneeContains string,
	priorityProp string,
	priorityEquals string,
) ([]map[string]interface{}, error) {
	needSchema := strings.TrimSpace(statusEquals) != "" || strings.TrimSpace(assigneeContains) != "" || strings.TrimSpace(priorityEquals) != ""
	if !needSchema {
		return nil, nil
	}
	if strings.TrimSpace(dataSourceID) == "" {
		return nil, fmt.Errorf("data source id is required to build shorthand filters")
	}

	ds, err := client.GetDataSource(ctx, dataSourceID)
	if err != nil {
		return nil, errors.WrapUserError(
			err,
			"failed to fetch data source schema for shorthand filters",
			"Try again, or provide a JSON filter via --filter/@file instead of shorthand flags.",
		)
	}

	props := map[string]interface{}{}
	if ds != nil && ds.Properties != nil {
		props = ds.Properties
	}

	var out []map[string]interface{}

	if strings.TrimSpace(statusEquals) != "" {
		propName := strings.TrimSpace(statusProp)
		propType, ok := findDataSourcePropertyType(props, propName)
		if !ok {
			return nil, errors.NewUserError(
				fmt.Sprintf("unknown property %q for --status", propName),
				"Use --status-prop to set the correct property name, or provide --filter JSON.",
			)
		}
		switch propType {
		case "status":
			out = append(out, map[string]interface{}{
				"property": propName,
				"status": map[string]interface{}{
					"equals": statusEquals,
				},
			})
		case "select":
			out = append(out, map[string]interface{}{
				"property": propName,
				"select": map[string]interface{}{
					"equals": statusEquals,
				},
			})
		default:
			return nil, errors.NewUserError(
				fmt.Sprintf("property %q is type %q; cannot use --status", propName, propType),
				"Use --filter JSON for advanced filtering, or pick a status/select property.",
			)
		}
	}

	if strings.TrimSpace(priorityEquals) != "" {
		propName := strings.TrimSpace(priorityProp)
		propType, ok := findDataSourcePropertyType(props, propName)
		if !ok {
			return nil, errors.NewUserError(
				fmt.Sprintf("unknown property %q for --priority", propName),
				"Use --priority-prop to set the correct property name, or provide --filter JSON.",
			)
		}
		switch propType {
		case "select":
			out = append(out, map[string]interface{}{
				"property": propName,
				"select": map[string]interface{}{
					"equals": priorityEquals,
				},
			})
		case "status":
			out = append(out, map[string]interface{}{
				"property": propName,
				"status": map[string]interface{}{
					"equals": priorityEquals,
				},
			})
		default:
			return nil, errors.NewUserError(
				fmt.Sprintf("property %q is type %q; cannot use --priority", propName, propType),
				"Use --filter JSON for advanced filtering, or pick a select/status property.",
			)
		}
	}

	if strings.TrimSpace(assigneeContains) != "" {
		propName := strings.TrimSpace(assigneeProp)
		propType, ok := findDataSourcePropertyType(props, propName)
		if !ok {
			return nil, errors.NewUserError(
				fmt.Sprintf("unknown property %q for --assignee", propName),
				"Use --assignee-prop to set the correct property name, or provide --filter JSON.",
			)
		}
		if propType != "people" {
			return nil, errors.NewUserError(
				fmt.Sprintf("property %q is type %q; cannot use --assignee", propName, propType),
				"Use --filter JSON for advanced filtering, or pick a people property.",
			)
		}

		assignee := strings.TrimSpace(assigneeContains)
		assignee = resolveUserID(sf, assignee)
		if !uuidLike.MatchString(assignee) {
			return nil, errors.NewUserError(
				fmt.Sprintf("invalid --assignee value %q (expected a user id or skill alias)", assigneeContains),
				"Run 'ntn skill init' to create user aliases, or pass a user UUID.",
			)
		}

		out = append(out, map[string]interface{}{
			"property": propName,
			"people": map[string]interface{}{
				"contains": strings.ToLower(assignee),
			},
		})
	}

	return out, nil
}

func mergeNotionFilters(base map[string]interface{}, extras []map[string]interface{}) map[string]interface{} {
	if len(extras) == 0 {
		return base
	}
	if base == nil && len(extras) == 1 {
		return extras[0]
	}

	and := make([]interface{}, 0, 1+len(extras))
	if base != nil {
		and = append(and, base)
	}
	for _, e := range extras {
		and = append(and, e)
	}
	return map[string]interface{}{
		"and": and,
	}
}

func findDataSourcePropertyType(props map[string]interface{}, name string) (string, bool) {
	if props == nil {
		return "", false
	}
	norm := normalizePropName(name)
	for k, v := range props {
		if normalizePropName(k) != norm {
			continue
		}
		m, ok := v.(map[string]interface{})
		if !ok {
			return "", false
		}
		t, _ := m["type"].(string)
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			return "", false
		}
		return t, true
	}
	return "", false
}

// normalizePropName normalizes a property name for fuzzy matching.
// Strips spaces, underscores, and lowercases so agents can write
// "due_date", "Due Date", or "duedate" and all match the same property.
// This is intentional for agent ergonomics — property names in Notion
// schemas vary in casing and separator style.
func normalizePropName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}
