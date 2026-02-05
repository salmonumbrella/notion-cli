package output

import "strings"

// NormalizeQuery removes common shell-escaped sequences that break jq parsing.
// Today this only removes "\!" outside string literals (bash history expansion).
// It returns the normalized query and whether a change was made.
func NormalizeQuery(query string) (string, bool) {
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
