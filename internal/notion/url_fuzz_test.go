package notion

import "testing"

func FuzzExtractIDFromNotionURL(f *testing.F) {
	f.Add("https://www.notion.so/Some-Page-12345678123412341234123456789012")
	f.Add("12345678-1234-1234-1234-123456789012")
	f.Add("notion.so/Some-Page-ABCDEFabcdef12345678901234567890")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		id, err := ExtractIDFromNotionURL(input)
		if err != nil {
			return
		}
		if !notionIDRegex.MatchString(id) {
			t.Fatalf("extracted id %q does not match Notion ID pattern", id)
		}
	})
}
