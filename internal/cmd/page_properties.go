package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/salmonumbrella/notion-cli/internal/richtext"
	"github.com/salmonumbrella/notion-cli/internal/skill"
)

func propertyHasValue(value interface{}) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case string:
		return v != ""
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
}

// transformPropertiesWithMentions transforms string shorthand values in properties
// to rich_text arrays with mentions. Only string values are transformed; other
// property types (arrays, objects) are passed through unchanged.
// Properties are processed in alphabetical order by name for deterministic
// user ID assignment when multiple properties contain @Name patterns.
// Returns the transformed properties and the number of user IDs that were actually used.
func transformPropertiesWithMentions(properties map[string]interface{}, userIDs []string) (map[string]interface{}, int) {
	return transformPropertiesWithMentionsVerbose(io.Discard, properties, userIDs, false, false)
}

// transformPropertiesWithMentionsVerbose is like transformPropertiesWithMentions but
// optionally prints verbose output about markdown parsing and mention matching.
// The w parameter specifies where verbose and warning output is written (typically os.Stderr in production).
// When emitWarnings is true, warnings about unused --mention flags are also written to w.
func transformPropertiesWithMentionsVerbose(w io.Writer, properties map[string]interface{}, userIDs []string, verbose bool, emitWarnings bool) (map[string]interface{}, int) {
	result := make(map[string]interface{}, len(properties))
	userIDIndex := 0

	// Sort property names for deterministic iteration order
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		value := properties[name]
		// Only transform string values (shorthand for rich_text)
		if strVal, ok := value.(string); ok {
			// Parse markdown once - used for both verbose output and building rich text
			tokens := richtext.ParseMarkdown(strVal)

			if verbose {
				_, _ = fmt.Fprintf(w, "Property %q:\n", name)
				summary := richtext.SummarizeTokens(tokens)
				_, _ = fmt.Fprintf(w, "  %s\n", richtext.FormatSummary(summary))
			}

			// Count @Name patterns in this string to consume the right number of user IDs
			mentionsNeeded := richtext.CountMentions(strVal)

			// Allocate user IDs for this property
			var propertyUserIDs []string
			if userIDIndex < len(userIDs) {
				end := userIDIndex + mentionsNeeded
				if end > len(userIDs) {
					end = len(userIDs)
				}
				propertyUserIDs = userIDs[userIDIndex:end]
				userIDIndex = end
			}

			if verbose && mentionsNeeded > 0 {
				richtext.FormatMentionMappingsIndented(w, strVal, propertyUserIDs, "  ")
			}

			// Build rich text array from pre-parsed tokens (avoids redundant parsing)
			richTextContent := richtext.BuildWithMentionsFromTokens(tokens, propertyUserIDs, nil)

			// Convert to the format expected by Notion API
			richTextArray := make([]interface{}, len(richTextContent))
			for i, rt := range richTextContent {
				rtMap := map[string]interface{}{
					"type": rt.Type,
				}
				if rt.Text != nil {
					rtMap["text"] = map[string]interface{}{
						"content": rt.Text.Content,
					}
				}
				if rt.Mention != nil {
					mentionMap := map[string]interface{}{
						"type": rt.Mention.Type,
					}
					if rt.Mention.User != nil {
						mentionMap["user"] = map[string]interface{}{
							"id": rt.Mention.User.ID,
						}
					}
					rtMap["mention"] = mentionMap
				}
				if rt.Annotations != nil {
					rtMap["annotations"] = map[string]interface{}{
						"bold":          rt.Annotations.Bold,
						"italic":        rt.Annotations.Italic,
						"strikethrough": rt.Annotations.Strikethrough,
						"underline":     rt.Annotations.Underline,
						"code":          rt.Annotations.Code,
						"color":         rt.Annotations.Color,
					}
				}
				richTextArray[i] = rtMap
			}

			// Wrap in rich_text property structure
			result[name] = map[string]interface{}{
				"rich_text": richTextArray,
			}
		} else {
			// Pass through non-string values unchanged
			result[name] = value
		}
	}

	// Emit warnings about unused --mention flags if requested
	if emitWarnings && len(userIDs) > 0 {
		if userIDIndex == 0 {
			_, _ = fmt.Fprintf(w, "warning: %d --mention flag(s) provided but no @Name patterns found in property values\n", len(userIDs))
		} else if userIDIndex < len(userIDs) {
			_, _ = fmt.Fprintf(w, "warning: %d of %d --mention flag(s) unused (not enough @Name patterns)\n", len(userIDs)-userIDIndex, len(userIDs))
		}
	}

	return result, userIDIndex
}

// buildPropertiesFromFlags merges shorthand property flags into a properties map.
// The shorthand flags take precedence over properties already in the map.
// If properties is nil, a new map is created.
// Supports: --status (status property), --priority (select property), --assignee (people property).
func buildPropertiesFromFlags(sf *skill.SkillFile, properties map[string]interface{}, status, priority, assignee string) map[string]interface{} {
	if properties == nil {
		properties = make(map[string]interface{})
	}

	if status != "" {
		properties["Status"] = map[string]interface{}{
			"status": map[string]interface{}{
				"name": status,
			},
		}
	}

	if priority != "" {
		properties["Priority"] = map[string]interface{}{
			"select": map[string]interface{}{
				"name": priority,
			},
		}
	}

	if assignee != "" {
		resolvedID := resolveUserID(sf, assignee)
		properties["Assignee"] = map[string]interface{}{
			"people": []map[string]interface{}{
				{"object": "user", "id": resolvedID},
			},
		}
	}

	return properties
}

// findTitlePropertyName finds the title property name in a database schema.
// Returns "title" as the default if no title property is found.
func findTitlePropertyName(properties map[string]map[string]interface{}) string {
	for propName, propDef := range properties {
		if propType, ok := propDef["type"].(string); ok && propType == "title" {
			return propName
		}
	}
	return "title"
}

// findTitlePropertyNameFromPage finds the title property name from a page's properties.
// Returns "title" as the default if no title property is found.
func findTitlePropertyNameFromPage(properties map[string]interface{}) string {
	for propName, propVal := range properties {
		prop, ok := propVal.(map[string]interface{})
		if !ok {
			continue
		}
		if prop["type"] == "title" {
			return propName
		}
	}
	return "title"
}

// findTitlePropertyNameFromDataSource finds the title property name from a data source schema.
// Returns "title" as the default if no title property is found.
func findTitlePropertyNameFromDataSource(properties map[string]interface{}) string {
	for propName, propVal := range properties {
		prop, ok := propVal.(map[string]interface{})
		if !ok {
			continue
		}
		if propType, ok := prop["type"].(string); ok && propType == "title" {
			return propName
		}
	}
	return "title"
}

// setTitleProperty sets the title property in a properties map using the given property name.
// The title is formatted as a rich text array with a single text element.
func setTitleProperty(properties map[string]interface{}, propName, title string) map[string]interface{} {
	if properties == nil {
		properties = make(map[string]interface{})
	}
	properties[propName] = map[string]interface{}{
		"title": []map[string]interface{}{
			{
				"text": map[string]interface{}{
					"content": title,
				},
			},
		},
	}
	return properties
}
