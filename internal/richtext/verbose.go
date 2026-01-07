package richtext

import (
	"fmt"
	"io"
)

// FormatMentionMappings writes mention-to-userID mappings to w.
// Output format:
//
//	Mentions:
//	  @Name -> user-id
//	  @Other -> (no user ID provided)
//
// If there are no mentions in the text, nothing is written.
func FormatMentionMappings(w io.Writer, text string, userIDs []string) {
	mentions := FindMentions(text)
	if len(mentions) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "Mentions:")
	for i, mention := range mentions {
		if i < len(userIDs) {
			_, _ = fmt.Fprintf(w, "  %s -> %s\n", mention, userIDs[i])
		} else {
			_, _ = fmt.Fprintf(w, "  %s -> (no user ID provided)\n", mention)
		}
	}
}

// FormatMentionMappingsIndented is like FormatMentionMappings but prefixes each line with indent.
func FormatMentionMappingsIndented(w io.Writer, text string, userIDs []string, indent string) {
	mentions := FindMentions(text)
	if len(mentions) == 0 {
		return
	}

	_, _ = fmt.Fprintf(w, "%sMentions:\n", indent)
	for i, mention := range mentions {
		if i < len(userIDs) {
			_, _ = fmt.Fprintf(w, "%s  %s -> %s\n", indent, mention, userIDs[i])
		} else {
			_, _ = fmt.Fprintf(w, "%s  %s -> (no user ID provided)\n", indent, mention)
		}
	}
}
