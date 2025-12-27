package output

import "strings"

type pathAliasSpec struct {
	Canonical string
	Aliases   []string
}

// pathAliasSpecs defines shorthand aliases for common JSON path keys used with
// --query/--jq, --fields/--pick, --jsonpath, and --sort-by.
//
// Aliases are intentionally lowercase. We only rewrite lowercase dot-path
// segments so mixed-case keys (for example Notion property names like "Status")
// are left untouched.
var pathAliasSpecs = []pathAliasSpec{
	{Canonical: "properties", Aliases: []string{"props", "pr"}},
	{Canonical: "rich_text", Aliases: []string{"rt"}},
	{Canonical: "plain_text", Aliases: []string{"pt", "p"}},
	{Canonical: "results", Aliases: []string{"rs"}},
	{Canonical: "object", Aliases: []string{"ob"}},
	{Canonical: "parent", Aliases: []string{"pa"}},
	{Canonical: "children", Aliases: []string{"ch"}},
	{Canonical: "has_children", Aliases: []string{"hc"}},
	{Canonical: "created_time", Aliases: []string{"ct"}},
	{Canonical: "last_edited_time", Aliases: []string{"lt"}},
	{Canonical: "created_by", Aliases: []string{"cb"}},
	{Canonical: "last_edited_by", Aliases: []string{"lb"}},
	{Canonical: "archived", Aliases: []string{"ar"}},
	{Canonical: "in_trash", Aliases: []string{"it"}},
	{Canonical: "public_url", Aliases: []string{"pu"}},
	{Canonical: "data_sources", Aliases: []string{"ds"}},
	{Canonical: "data_source_id", Aliases: []string{"dsi"}},
	{Canonical: "database_id", Aliases: []string{"dbi"}},
	{Canonical: "page_id", Aliases: []string{"pid"}},
	{Canonical: "workspace_id", Aliases: []string{"wid"}},
	{Canonical: "discussion_id", Aliases: []string{"did"}},
	{Canonical: "comment_id", Aliases: []string{"cid"}},
	{Canonical: "parent_title", Aliases: []string{"ptt"}},
	{Canonical: "child_count", Aliases: []string{"cc"}},
	{Canonical: "next_cursor", Aliases: []string{"nc"}},
	{Canonical: "has_more", Aliases: []string{"hm"}},
	{Canonical: "start_cursor", Aliases: []string{"sc"}},
	{Canonical: "page_size", Aliases: []string{"ps"}},
	{Canonical: "sorts", Aliases: []string{"so"}},
	{Canonical: "filter", Aliases: []string{"fi"}},
	{Canonical: "query", Aliases: []string{"qy"}},
	{Canonical: "multi_select", Aliases: []string{"ms"}},
	{Canonical: "phone_number", Aliases: []string{"ph"}},
	{Canonical: "time_zone", Aliases: []string{"tz"}},
	{Canonical: "unique_id", Aliases: []string{"uid"}},
	{Canonical: "upload_url", Aliases: []string{"uu"}},
	{Canonical: "expiry_time", Aliases: []string{"et"}},
	{Canonical: "file_name", Aliases: []string{"fn"}},
	{Canonical: "mime_type", Aliases: []string{"mt"}},
	{Canonical: "is_inline", Aliases: []string{"ii"}},
	{Canonical: "initial_data_source", Aliases: []string{"ids"}},
	{Canonical: "verification_token", Aliases: []string{"vt"}},
	{Canonical: "_meta", Aliases: []string{"meta"}},
	{Canonical: "status", Aliases: []string{"st"}},
	{Canonical: "select", Aliases: []string{"sl"}},
	{Canonical: "relation", Aliases: []string{"rl"}},
	{Canonical: "people", Aliases: []string{"pe"}},
	{Canonical: "checkbox", Aliases: []string{"cbx"}},
	{Canonical: "number", Aliases: []string{"nu"}},
	{Canonical: "files", Aliases: []string{"fl"}},
	{Canonical: "content", Aliases: []string{"co"}},
	{Canonical: "text", Aliases: []string{"tx"}},
	{Canonical: "title", Aliases: []string{"ti", "t"}},
	{Canonical: "name", Aliases: []string{"nm"}},
	{Canonical: "type", Aliases: []string{"ty"}},
	{Canonical: "url", Aliases: []string{"ur"}},
	{Canonical: "cover", Aliases: []string{"cv"}},
	{Canonical: "icon", Aliases: []string{"ic"}},
}

var pathAliasLookup = buildPathAliasLookup()

func buildPathAliasLookup() map[string]string {
	out := make(map[string]string)
	for _, spec := range pathAliasSpecs {
		canonical := strings.TrimSpace(spec.Canonical)
		if canonical == "" {
			continue
		}
		for _, alias := range spec.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			if existing, ok := out[alias]; ok && existing != canonical {
				panic("duplicate path alias: " + alias)
			}
			out[alias] = canonical
		}
	}
	return out
}

func canonicalizeAliasToken(token string) string {
	if token == "" {
		return token
	}
	// Keep behavior predictable: only rewrite lowercase tokens.
	if token != strings.ToLower(token) {
		return token
	}
	if canonical, ok := pathAliasLookup[token]; ok {
		return canonical
	}
	return token
}

func isAliasIdentifierStart(ch byte) bool {
	return ch == '_' ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z')
}

func isAliasIdentifierPart(ch byte) bool {
	return isAliasIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

// expandDotPathAliases rewrites lowercase dot-path segments to their canonical
// names. Example: ".props.Name.rt[0].pt" -> ".properties.Name.rich_text[0].plain_text".
//
// String literals and comments are preserved verbatim.
func expandDotPathAliases(expr string) (string, bool) {
	if strings.TrimSpace(expr) == "" {
		return expr, false
	}

	var b strings.Builder
	b.Grow(len(expr))

	changed := false
	inDouble := false
	inSingle := false
	escaped := false
	inComment := false

	for i := 0; i < len(expr); i++ {
		ch := expr[i]

		if inComment {
			b.WriteByte(ch)
			if ch == '\n' {
				inComment = false
			}
			continue
		}

		if inDouble {
			b.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inDouble = false
			}
			continue
		}

		// jq uses only double-quoted strings, but we handle single quotes
		// defensively for non-jq contexts (JSONPath, --fields values) and
		// to guard against shell quoting edge cases.
		if inSingle {
			b.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '\'' {
				inSingle = false
			}
			continue
		}

		switch ch {
		case '#':
			inComment = true
			b.WriteByte(ch)
			continue
		case '"':
			inDouble = true
			b.WriteByte(ch)
			continue
		case '\'':
			inSingle = true
			b.WriteByte(ch)
			continue
		case '.':
			b.WriteByte(ch)
			if i+1 < len(expr) && isAliasIdentifierStart(expr[i+1]) {
				start := i + 1
				end := start + 1
				for end < len(expr) && isAliasIdentifierPart(expr[end]) {
					end++
				}
				ident := expr[start:end]
				canonical := canonicalizeAliasToken(ident)
				if canonical != ident {
					changed = true
				}
				b.WriteString(canonical)
				i = end - 1
				continue
			}
			continue
		}

		b.WriteByte(ch)
	}

	if !changed {
		return expr, false
	}
	return b.String(), true
}

// NormalizeSortPath rewrites dot-path aliases for --sort-by.
// Example: "ct" -> "created_time", "props.Name.ct" -> "properties.Name.created_time".
func NormalizeSortPath(path string) (string, bool) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return trimmed, false
	}

	parts := strings.Split(trimmed, ".")
	changed := false
	for i, part := range parts {
		if part == "" {
			continue
		}
		canonical := canonicalizeAliasToken(part)
		if canonical != part {
			parts[i] = canonical
			changed = true
		}
	}

	if !changed {
		return trimmed, false
	}
	return strings.Join(parts, "."), true
}
