package richtext

import (
	"fmt"
	"io"
)

// FormatMentionMappings writes mention-to-userID mappings to w.
// Output format:
//
//	Mentions:
//	  @Name → user-id
//	  @Other → (no user ID available)
//
// If there are no mentions in the text, nothing is written.
func FormatMentionMappings(w io.Writer, text string, userIDs []string) {
	FormatAllMentionMappings(w, text, userIDs, nil)
}

// FormatAllMentionMappings writes both user (@) and page (@@) mention mappings to w.
// Output format:
//
//	User mentions:
//	  @Name → user-id
//	  @Other → (no user ID available)
//	Page mentions:
//	  @@PageName → page-id
//	  @@OtherPage → (no page ID available)
//
// If there are no mentions of either type, nothing is written for that type.
func FormatAllMentionMappings(w io.Writer, text string, userIDs []string, pageIDs []string) {
	userMentions := FindMentions(text)
	pageMentions := FindPageMentions(text)

	// Filter user mentions that are part of page mentions
	filteredUserMentions := filterUserMentionStrings(userMentions, pageMentions)

	if len(filteredUserMentions) > 0 {
		_, _ = fmt.Fprintln(w, "User mentions:")
		for i, mention := range filteredUserMentions {
			if i < len(userIDs) {
				_, _ = fmt.Fprintf(w, "  %s → %s\n", mention, userIDs[i])
			} else {
				_, _ = fmt.Fprintf(w, "  %s → (no user ID available)\n", mention)
			}
		}
	}

	if len(pageMentions) > 0 {
		_, _ = fmt.Fprintln(w, "Page mentions:")
		for i, mention := range pageMentions {
			if i < len(pageIDs) {
				_, _ = fmt.Fprintf(w, "  %s → %s\n", mention, pageIDs[i])
			} else {
				_, _ = fmt.Fprintf(w, "  %s → (no page ID available)\n", mention)
			}
		}
	}
}

// filterUserMentionStrings removes user mentions that are part of page mentions.
// For example, if we have "@@Page", the @Page part should not be counted as a user mention.
func filterUserMentionStrings(userMentions, pageMentions []string) []string {
	if len(pageMentions) == 0 {
		return userMentions
	}

	// Build a set of user mentions that are part of page mentions
	// @@Name contains @Name, so @Name would be "@@Name"[1:]
	partOfPage := make(map[string]bool)
	for _, pm := range pageMentions {
		if len(pm) > 1 {
			// Remove one @ to get the embedded user mention pattern
			partOfPage[pm[1:]] = true
		}
	}

	var filtered []string
	for _, um := range userMentions {
		if !partOfPage[um] {
			filtered = append(filtered, um)
		}
	}
	return filtered
}

// FormatMentionMappingsIndented is like FormatMentionMappings but prefixes each line with indent.
func FormatMentionMappingsIndented(w io.Writer, text string, userIDs []string, indent string) {
	FormatAllMentionMappingsIndented(w, text, userIDs, nil, indent)
}

// FormatAllMentionMappingsIndented is like FormatAllMentionMappings but prefixes each line with indent.
func FormatAllMentionMappingsIndented(w io.Writer, text string, userIDs []string, pageIDs []string, indent string) {
	userMentions := FindMentions(text)
	pageMentions := FindPageMentions(text)

	// Filter user mentions that are part of page mentions
	filteredUserMentions := filterUserMentionStrings(userMentions, pageMentions)

	if len(filteredUserMentions) > 0 {
		_, _ = fmt.Fprintf(w, "%sUser mentions:\n", indent)
		for i, mention := range filteredUserMentions {
			if i < len(userIDs) {
				_, _ = fmt.Fprintf(w, "%s  %s → %s\n", indent, mention, userIDs[i])
			} else {
				_, _ = fmt.Fprintf(w, "%s  %s → (no user ID available)\n", indent, mention)
			}
		}
	}

	if len(pageMentions) > 0 {
		_, _ = fmt.Fprintf(w, "%sPage mentions:\n", indent)
		for i, mention := range pageMentions {
			if i < len(pageIDs) {
				_, _ = fmt.Fprintf(w, "%s  %s → %s\n", indent, mention, pageIDs[i])
			} else {
				_, _ = fmt.Fprintf(w, "%s  %s → (no page ID available)\n", indent, mention)
			}
		}
	}
}

// FormatLinkWarnings writes link URL validation warnings to w.
// Does nothing if warnings slice is empty.
func FormatLinkWarnings(w io.Writer, warnings []string) {
	if len(warnings) == 0 {
		return
	}
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(w, "warning: %s\n", warning)
	}
}
