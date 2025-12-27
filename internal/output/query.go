package output

import "strings"

// NormalizeQuery normalizes jq query input.
// It:
// - removes shell-escaped "\!" outside string literals
// - expands supported shorthand dot-path aliases (for example .props/.rt/.pt)
//
// The returned bool is true only when "\!" normalization was applied. We use
// that signal for an interactive warning about shell escaping.
func NormalizeQuery(query string) (string, bool) {
	normalized, escapedBangChanged := normalizeEscapedBang(query)
	normalized, _ = expandDotPathAliases(normalized)
	return normalized, escapedBangChanged
}

func normalizeEscapedBang(query string) (string, bool) {
	if !strings.Contains(query, `\!`) {
		return query, false
	}

	var b strings.Builder
	b.Grow(len(query))

	inString := false
	escaped := false
	changed := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		if inString {
			if escaped {
				escaped = false
				b.WriteByte(ch)
				continue
			}
			if ch == '\\' {
				escaped = true
				b.WriteByte(ch)
				continue
			}
			if ch == '"' {
				inString = false
			}
			b.WriteByte(ch)
			continue
		}

		if ch == '"' {
			inString = true
			b.WriteByte(ch)
			continue
		}

		if ch == '\\' && i+1 < len(query) && query[i+1] == '!' {
			changed = true
			b.WriteByte('!')
			i++
			continue
		}

		b.WriteByte(ch)
	}

	if !changed {
		return query, false
	}
	return b.String(), true
}
