package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/skill"
)

func TestPropertyHasValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  bool
	}{
		// nil cases
		{
			name:  "nil returns false",
			value: nil,
			want:  false,
		},

		// string cases
		{
			name:  "empty string returns false",
			value: "",
			want:  false,
		},
		{
			name:  "non-empty string returns true",
			value: "hello",
			want:  true,
		},
		{
			name:  "whitespace-only string returns true",
			value: "   ",
			want:  true,
		},

		// slice cases
		{
			name:  "empty slice returns false",
			value: []interface{}{},
			want:  false,
		},
		{
			name:  "non-empty slice returns true",
			value: []interface{}{"item"},
			want:  true,
		},
		{
			name:  "slice with nil element returns true",
			value: []interface{}{nil},
			want:  true,
		},
		{
			name:  "slice with multiple elements returns true",
			value: []interface{}{"a", "b", "c"},
			want:  true,
		},

		// map cases
		{
			name:  "empty map returns false",
			value: map[string]interface{}{},
			want:  false,
		},
		{
			name:  "non-empty map returns true",
			value: map[string]interface{}{"key": "value"},
			want:  true,
		},
		{
			name:  "map with nil value returns true",
			value: map[string]interface{}{"key": nil},
			want:  true,
		},

		// bool cases - intentionally return true for false (checkbox "unchecked" is still "set")
		{
			name:  "bool false returns true",
			value: false,
			want:  true,
		},
		{
			name:  "bool true returns true",
			value: true,
			want:  true,
		},

		// int cases - intentionally return true for 0 (number property set to 0 is still "set")
		{
			name:  "int 0 returns true",
			value: 0,
			want:  true,
		},
		{
			name:  "positive int returns true",
			value: 42,
			want:  true,
		},
		{
			name:  "negative int returns true",
			value: -10,
			want:  true,
		},

		// float cases - intentionally return true for 0.0
		{
			name:  "float 0.0 returns true",
			value: 0.0,
			want:  true,
		},
		{
			name:  "positive float returns true",
			value: 3.14,
			want:  true,
		},
		{
			name:  "negative float returns true",
			value: -2.5,
			want:  true,
		},

		// other types that should return true via default case
		{
			name:  "struct returns true",
			value: struct{ Name string }{Name: "test"},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := propertyHasValue(tt.value)
			if got != tt.want {
				t.Errorf("propertyHasValue(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestTransformPropertiesWithMentions(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]interface{}
		userIDs    []string
		// We check specific aspects of the result rather than exact match
		checkFunc func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "string value with mention is transformed to rich_text",
			properties: map[string]interface{}{
				"Summary": "@Reviewer should film this",
			},
			userIDs: []string{"reviewer-user-id"},
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				summary, ok := result["Summary"].(map[string]interface{})
				if !ok {
					t.Fatalf("Summary should be a map, got %T", result["Summary"])
				}
				richText, ok := summary["rich_text"].([]interface{})
				if !ok {
					t.Fatalf("Summary should have rich_text array, got %T", summary["rich_text"])
				}
				if len(richText) != 2 {
					t.Fatalf("expected 2 rich_text elements, got %d", len(richText))
				}
				// First element should be a mention
				first := richText[0].(map[string]interface{})
				if first["type"] != "mention" {
					t.Errorf("first element should be mention, got %v", first["type"])
				}
				mention := first["mention"].(map[string]interface{})
				user := mention["user"].(map[string]interface{})
				if user["id"] != "reviewer-user-id" {
					t.Errorf("expected user id reviewer-user-id, got %v", user["id"])
				}
				// Second element should be text
				second := richText[1].(map[string]interface{})
				if second["type"] != "text" {
					t.Errorf("second element should be text, got %v", second["type"])
				}
				text := second["text"].(map[string]interface{})
				if text["content"] != " should film this" {
					t.Errorf("expected ' should film this', got %v", text["content"])
				}
			},
		},
		{
			name: "multiple mentions across multiple properties - alphabetical order",
			properties: map[string]interface{}{
				"Summary": "@Second review",
				"Notes":   "@First check",
			},
			userIDs: []string{"first-id", "second-id"},
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				// Properties are processed alphabetically, so Notes comes before Summary.
				// first-id should be assigned to Notes, second-id to Summary.

				// Check Notes got first-id
				notes := result["Notes"].(map[string]interface{})
				notesRT := notes["rich_text"].([]interface{})
				notesMention := notesRT[0].(map[string]interface{})
				if notesMention["type"] != "mention" {
					t.Fatalf("Notes first element should be mention")
				}
				notesMentionData := notesMention["mention"].(map[string]interface{})
				notesUser := notesMentionData["user"].(map[string]interface{})
				if notesUser["id"] != "first-id" {
					t.Errorf("Notes should get first-id (alphabetically first), got %v", notesUser["id"])
				}

				// Check Summary got second-id
				summary := result["Summary"].(map[string]interface{})
				summaryRT := summary["rich_text"].([]interface{})
				summaryMention := summaryRT[0].(map[string]interface{})
				if summaryMention["type"] != "mention" {
					t.Fatalf("Summary first element should be mention")
				}
				summaryMentionData := summaryMention["mention"].(map[string]interface{})
				summaryUser := summaryMentionData["user"].(map[string]interface{})
				if summaryUser["id"] != "second-id" {
					t.Errorf("Summary should get second-id (alphabetically second), got %v", summaryUser["id"])
				}
			},
		},
		{
			name: "non-string values pass through unchanged",
			properties: map[string]interface{}{
				"Status": map[string]interface{}{
					"select": map[string]interface{}{
						"name": "In Progress",
					},
				},
				"Count": float64(42),
			},
			userIDs: []string{"unused-id"},
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				// Status should be passed through as-is
				status := result["Status"].(map[string]interface{})
				sel := status["select"].(map[string]interface{})
				if sel["name"] != "In Progress" {
					t.Errorf("Status select should be unchanged")
				}
				// Count should be passed through as-is
				if result["Count"] != float64(42) {
					t.Errorf("Count should be 42, got %v", result["Count"])
				}
			},
		},
		{
			name: "array property values pass through unchanged",
			properties: map[string]interface{}{
				"Related Pages": map[string]interface{}{
					"relation": []interface{}{
						map[string]interface{}{"id": "page-1"},
						map[string]interface{}{"id": "page-2"},
					},
				},
				"Summary": "@Alice check these",
			},
			userIDs: []string{"alice-id"},
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				// Relation array should be passed through as-is
				relatedPages := result["Related Pages"].(map[string]interface{})
				relation := relatedPages["relation"].([]interface{})
				if len(relation) != 2 {
					t.Fatalf("expected 2 relation items, got %d", len(relation))
				}
				first := relation[0].(map[string]interface{})
				if first["id"] != "page-1" {
					t.Errorf("first relation id should be page-1, got %v", first["id"])
				}
				second := relation[1].(map[string]interface{})
				if second["id"] != "page-2" {
					t.Errorf("second relation id should be page-2, got %v", second["id"])
				}
				// Summary should still be transformed with mention
				summary := result["Summary"].(map[string]interface{})
				richText := summary["rich_text"].([]interface{})
				if len(richText) != 2 {
					t.Fatalf("expected 2 rich_text elements, got %d", len(richText))
				}
				firstRT := richText[0].(map[string]interface{})
				if firstRT["type"] != "mention" {
					t.Errorf("first element should be mention, got %v", firstRT["type"])
				}
			},
		},
		{
			name: "string without mention still transforms to rich_text",
			properties: map[string]interface{}{
				"Description": "Plain text here",
			},
			userIDs: []string{"unused-id"},
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				desc := result["Description"].(map[string]interface{})
				richText := desc["rich_text"].([]interface{})
				if len(richText) != 1 {
					t.Fatalf("expected 1 rich_text element, got %d", len(richText))
				}
				first := richText[0].(map[string]interface{})
				if first["type"] != "text" {
					t.Errorf("expected text type, got %v", first["type"])
				}
				text := first["text"].(map[string]interface{})
				if text["content"] != "Plain text here" {
					t.Errorf("expected 'Plain text here', got %v", text["content"])
				}
			},
		},
		{
			name: "markdown formatting is preserved",
			properties: map[string]interface{}{
				"Notes": "This is **bold** text",
			},
			userIDs: nil,
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				notes := result["Notes"].(map[string]interface{})
				richText := notes["rich_text"].([]interface{})
				if len(richText) != 3 {
					t.Fatalf("expected 3 rich_text elements, got %d", len(richText))
				}
				// Middle element should be bold
				middle := richText[1].(map[string]interface{})
				annotations := middle["annotations"].(map[string]interface{})
				if annotations["bold"] != true {
					t.Errorf("expected bold annotation to be true")
				}
			},
		},
		{
			name: "more @Names than user IDs - extras kept as text",
			properties: map[string]interface{}{
				"Summary": "@Alice and @Bob and @Charlie",
			},
			userIDs: []string{"alice-id"},
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				summary := result["Summary"].(map[string]interface{})
				richText := summary["rich_text"].([]interface{})
				// Should have: mention, text(" and "), text(@Bob), text(" and "), text(@Charlie)
				mentionCount := 0
				textCount := 0
				for _, rt := range richText {
					rtMap := rt.(map[string]interface{})
					if rtMap["type"] == "mention" {
						mentionCount++
					} else {
						textCount++
					}
				}
				if mentionCount != 1 {
					t.Errorf("expected 1 mention, got %d", mentionCount)
				}
				if textCount != 4 {
					t.Errorf("expected 4 text elements, got %d", textCount)
				}
			},
		},
		{
			name: "empty string with mentions - transforms to empty rich_text, user IDs unused",
			properties: map[string]interface{}{
				"Summary": "",
			},
			userIDs: []string{"unused-user-id"},
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				summary := result["Summary"].(map[string]interface{})
				richText, ok := summary["rich_text"].([]interface{})
				if !ok {
					t.Fatalf("Summary should have rich_text array, got %T", summary["rich_text"])
				}
				// Empty string should produce empty rich_text array
				if len(richText) != 0 {
					t.Errorf("expected empty rich_text array, got %d elements", len(richText))
				}
				// User ID is unused (silently) - this documents the current behavior
				// Note: user IDs are only consumed when @Name patterns are found
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := transformPropertiesWithMentions(tt.properties, tt.userIDs)
			tt.checkFunc(t, result)
		})
	}
}

func TestTransformPropertiesWithMentions_UsedCount(t *testing.T) {
	tests := []struct {
		name         string
		properties   map[string]interface{}
		userIDs      []string
		expectedUsed int
	}{
		{
			name:         "all user IDs used",
			properties:   map[string]interface{}{"Summary": "@Alice and @Bob"},
			userIDs:      []string{"alice-id", "bob-id"},
			expectedUsed: 2,
		},
		{
			name:         "some user IDs unused",
			properties:   map[string]interface{}{"Summary": "@Alice only"},
			userIDs:      []string{"alice-id", "bob-id", "charlie-id"},
			expectedUsed: 1,
		},
		{
			name:         "no @Name patterns - zero used",
			properties:   map[string]interface{}{"Summary": "plain text"},
			userIDs:      []string{"unused-id"},
			expectedUsed: 0,
		},
		{
			name:         "empty string - zero used",
			properties:   map[string]interface{}{"Summary": ""},
			userIDs:      []string{"unused-id"},
			expectedUsed: 0,
		},
		{
			name:         "non-string values - zero used",
			properties:   map[string]interface{}{"Count": float64(42)},
			userIDs:      []string{"unused-id"},
			expectedUsed: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, usedCount := transformPropertiesWithMentions(tt.properties, tt.userIDs)
			if usedCount != tt.expectedUsed {
				t.Errorf("expected %d user IDs used, got %d", tt.expectedUsed, usedCount)
			}
		})
	}
}

func TestTransformPropertiesWithMentionsVerbose(t *testing.T) {
	// Test that verbose=true produces the same results as verbose=false
	properties := map[string]interface{}{
		"Summary": "@Reviewer should **review** this",
		"Notes":   "Plain text",
	}
	userIDs := []string{"reviewer-user-id"}

	resultVerbose, usedVerbose := transformPropertiesWithMentionsVerbose(io.Discard, properties, userIDs, true, false)
	resultNonVerbose, usedNonVerbose := transformPropertiesWithMentionsVerbose(io.Discard, properties, userIDs, false, false)

	// Used counts should be the same
	if usedVerbose != usedNonVerbose {
		t.Errorf("verbose and non-verbose should have same used count, got %d vs %d", usedVerbose, usedNonVerbose)
	}

	// Results should serialize to the same JSON
	verboseJSON, _ := json.Marshal(resultVerbose)
	nonVerboseJSON, _ := json.Marshal(resultNonVerbose)
	if string(verboseJSON) != string(nonVerboseJSON) {
		t.Errorf("verbose and non-verbose should produce same result")
	}
}

func TestTransformPropertiesWithMentionsVerbose_Output(t *testing.T) {
	// Test that verbose output contains expected content
	properties := map[string]interface{}{
		"Summary": "@Reviewer should **review** this",
	}
	userIDs := []string{"reviewer-user-id"}

	var buf bytes.Buffer
	_, _ = transformPropertiesWithMentionsVerbose(&buf, properties, userIDs, true, false)

	output := buf.String()
	if !strings.Contains(output, `Property "Summary"`) {
		t.Errorf("verbose output should contain property name, got: %s", output)
	}
	// Check that mention assignments are shown
	if !strings.Contains(output, "User mentions:") {
		t.Errorf("verbose output should contain 'User mentions:' header, got: %s", output)
	}
	if !strings.Contains(output, "@Reviewer → reviewer-user-id") {
		t.Errorf("verbose output should show @Name → user ID mapping, got: %s", output)
	}
}

func TestTransformPropertiesWithMentionsVerbose_NoUserID(t *testing.T) {
	// Test that verbose output shows when no user ID is available
	properties := map[string]interface{}{
		"Summary": "@Alice and @Bob",
	}
	userIDs := []string{"alice-id"} // Only one user ID for two mentions

	var buf bytes.Buffer
	_, _ = transformPropertiesWithMentionsVerbose(&buf, properties, userIDs, true, false)

	output := buf.String()
	if !strings.Contains(output, "@Alice → alice-id") {
		t.Errorf("verbose output should show @Alice mapped to alice-id, got: %s", output)
	}
	if !strings.Contains(output, "@Bob → (no user ID available)") {
		t.Errorf("verbose output should show @Bob with no user ID, got: %s", output)
	}
}

func TestTransformPropertiesWithMentionsVerbose_Warnings(t *testing.T) {
	tests := []struct {
		name           string
		properties     map[string]interface{}
		userIDs        []string
		emitWarnings   bool
		expectContains []string
		expectMissing  []string
	}{
		{
			name:         "no warning when emitWarnings is false",
			properties:   map[string]interface{}{"Summary": "plain text"},
			userIDs:      []string{"unused-id"},
			emitWarnings: false,
			expectMissing: []string{
				"warning:",
			},
		},
		{
			name:         "warning when all mentions unused",
			properties:   map[string]interface{}{"Summary": "plain text without any mentions"},
			userIDs:      []string{"unused-id-1", "unused-id-2"},
			emitWarnings: true,
			expectContains: []string{
				"warning: 2 --mention flag(s) provided but no @Name patterns found",
			},
		},
		{
			name:         "warning when some mentions unused",
			properties:   map[string]interface{}{"Summary": "@Alice only"},
			userIDs:      []string{"alice-id", "bob-id", "charlie-id"},
			emitWarnings: true,
			expectContains: []string{
				"warning: 2 of 3 --mention flag(s) unused",
			},
		},
		{
			name:         "no warning when all mentions used",
			properties:   map[string]interface{}{"Summary": "@Alice and @Bob"},
			userIDs:      []string{"alice-id", "bob-id"},
			emitWarnings: true,
			expectMissing: []string{
				"warning:",
			},
		},
		{
			name:         "no warning when no userIDs provided",
			properties:   map[string]interface{}{"Summary": "plain text"},
			userIDs:      nil,
			emitWarnings: true,
			expectMissing: []string{
				"warning:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			_, _ = transformPropertiesWithMentionsVerbose(&buf, tt.properties, tt.userIDs, false, tt.emitWarnings)

			output := buf.String()

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, got: %s", expected, output)
				}
			}

			for _, notExpected := range tt.expectMissing {
				if strings.Contains(output, notExpected) {
					t.Errorf("expected output to NOT contain %q, got: %s", notExpected, output)
				}
			}
		})
	}
}

func TestTransformPropertiesWithMentions_JSONOutput(t *testing.T) {
	// Test that the output can be serialized to valid JSON
	properties := map[string]interface{}{
		"Summary": "@Reviewer should review",
	}
	userIDs := []string{"reviewer-user-id"}

	result, _ := transformPropertiesWithMentions(properties, userIDs)

	// Should be serializable to JSON
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result to JSON: %v", err)
	}

	// Should be parseable back
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Verify structure
	summary := parsed["Summary"].(map[string]interface{})
	richText := summary["rich_text"].([]interface{})
	if len(richText) != 2 {
		t.Errorf("expected 2 rich_text elements after JSON roundtrip, got %d", len(richText))
	}
}

func TestTransformPropertiesWithMentions_RichTextFlagBehavior(t *testing.T) {
	// This test documents the behavior when --rich-text flag is used without --mention.
	// The transformation function is called with empty userIDs, and markdown should
	// still be parsed while @Name patterns remain as literal text.

	tests := []struct {
		name       string
		properties map[string]interface{}
		userIDs    []string // empty to simulate --rich-text without --mention
		checkFunc  func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "markdown parsed with empty userIDs (--rich-text flag behavior)",
			properties: map[string]interface{}{
				"Notes": "This is **bold** and *italic* text",
			},
			userIDs: nil, // No --mention flags, just --rich-text
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				notes := result["Notes"].(map[string]interface{})
				richText := notes["rich_text"].([]interface{})
				// Expected: "This is ", "bold" (bold), " and ", "italic" (italic), " text"
				if len(richText) != 5 {
					t.Fatalf("expected 5 rich_text elements, got %d", len(richText))
				}

				// Check bold element (index 1)
				boldElem := richText[1].(map[string]interface{})
				boldText := boldElem["text"].(map[string]interface{})
				if boldText["content"] != "bold" {
					t.Errorf("expected 'bold', got %v", boldText["content"])
				}
				boldAnnotations := boldElem["annotations"].(map[string]interface{})
				if boldAnnotations["bold"] != true {
					t.Errorf("expected bold annotation to be true")
				}

				// Check italic element (index 3)
				italicElem := richText[3].(map[string]interface{})
				italicText := italicElem["text"].(map[string]interface{})
				if italicText["content"] != "italic" {
					t.Errorf("expected 'italic', got %v", italicText["content"])
				}
				italicAnnotations := italicElem["annotations"].(map[string]interface{})
				if italicAnnotations["italic"] != true {
					t.Errorf("expected italic annotation to be true")
				}
			},
		},
		{
			name: "@Name patterns kept as text when no userIDs provided",
			properties: map[string]interface{}{
				"Summary": "@Alice should **review** this",
			},
			userIDs: nil, // No --mention flags
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				summary := result["Summary"].(map[string]interface{})
				richText := summary["rich_text"].([]interface{})

				// @Alice should remain as text (not a mention) since no userIDs provided
				// Check that there are no mention types
				for _, rt := range richText {
					rtMap := rt.(map[string]interface{})
					if rtMap["type"] == "mention" {
						t.Errorf("expected no mentions when userIDs is empty, but found one")
					}
				}

				// Check that **review** is still parsed as bold
				foundBold := false
				for _, rt := range richText {
					rtMap := rt.(map[string]interface{})
					if annotations, ok := rtMap["annotations"].(map[string]interface{}); ok {
						if annotations["bold"] == true {
							foundBold = true
							text := rtMap["text"].(map[string]interface{})
							if text["content"] != "review" {
								t.Errorf("expected bold text to be 'review', got %v", text["content"])
							}
						}
					}
				}
				if !foundBold {
					t.Errorf("expected to find bold 'review' text")
				}
			},
		},
		{
			name: "code formatting with --rich-text",
			properties: map[string]interface{}{
				"Code": "Use `fmt.Println` for output",
			},
			userIDs: nil,
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				code := result["Code"].(map[string]interface{})
				richText := code["rich_text"].([]interface{})

				// Find the code element
				foundCode := false
				for _, rt := range richText {
					rtMap := rt.(map[string]interface{})
					if annotations, ok := rtMap["annotations"].(map[string]interface{}); ok {
						if annotations["code"] == true {
							foundCode = true
							text := rtMap["text"].(map[string]interface{})
							if text["content"] != "fmt.Println" {
								t.Errorf("expected code text to be 'fmt.Println', got %v", text["content"])
							}
						}
					}
				}
				if !foundCode {
					t.Errorf("expected to find code-formatted 'fmt.Println'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := transformPropertiesWithMentions(tt.properties, tt.userIDs)
			tt.checkFunc(t, result)
		})
	}
}

func TestBuildPropertiesFromFlags(t *testing.T) {
	tests := []struct {
		name       string
		sf         *skill.SkillFile
		properties map[string]interface{}
		status     string
		priority   string
		assignee   string
		checkFunc  func(t *testing.T, result map[string]interface{})
	}{
		{
			name:       "nil properties creates new map",
			sf:         nil,
			properties: nil,
			status:     "Done",
			priority:   "",
			assignee:   "",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				if result == nil {
					t.Fatal("expected non-nil result")
				}
				status := result["Status"].(map[string]interface{})
				statusData := status["status"].(map[string]interface{})
				if statusData["name"] != "Done" {
					t.Errorf("expected Status name 'Done', got %v", statusData["name"])
				}
			},
		},
		{
			name:       "status flag sets Status property",
			sf:         nil,
			properties: make(map[string]interface{}),
			status:     "In Progress",
			priority:   "",
			assignee:   "",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				status := result["Status"].(map[string]interface{})
				statusData := status["status"].(map[string]interface{})
				if statusData["name"] != "In Progress" {
					t.Errorf("expected Status name 'In Progress', got %v", statusData["name"])
				}
			},
		},
		{
			name:       "priority flag sets Priority property",
			sf:         nil,
			properties: make(map[string]interface{}),
			status:     "",
			priority:   "High",
			assignee:   "",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				priority := result["Priority"].(map[string]interface{})
				selectData := priority["select"].(map[string]interface{})
				if selectData["name"] != "High" {
					t.Errorf("expected Priority name 'High', got %v", selectData["name"])
				}
			},
		},
		{
			name:       "assignee flag sets Assignee property with user ID",
			sf:         nil,
			properties: make(map[string]interface{}),
			status:     "",
			priority:   "",
			assignee:   "user-123",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				assignee := result["Assignee"].(map[string]interface{})
				people := assignee["people"].([]map[string]interface{})
				if len(people) != 1 {
					t.Fatalf("expected 1 person, got %d", len(people))
				}
				if people[0]["id"] != "user-123" {
					t.Errorf("expected id 'user-123', got %v", people[0]["id"])
				}
				if people[0]["object"] != "user" {
					t.Errorf("expected object 'user', got %v", people[0]["object"])
				}
			},
		},
		{
			name:       "all flags together",
			sf:         nil,
			properties: make(map[string]interface{}),
			status:     "Done",
			priority:   "Low",
			assignee:   "user-456",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				// Check Status
				status := result["Status"].(map[string]interface{})
				statusData := status["status"].(map[string]interface{})
				if statusData["name"] != "Done" {
					t.Errorf("expected Status name 'Done', got %v", statusData["name"])
				}

				// Check Priority
				priority := result["Priority"].(map[string]interface{})
				selectData := priority["select"].(map[string]interface{})
				if selectData["name"] != "Low" {
					t.Errorf("expected Priority name 'Low', got %v", selectData["name"])
				}

				// Check Assignee
				assignee := result["Assignee"].(map[string]interface{})
				people := assignee["people"].([]map[string]interface{})
				if people[0]["id"] != "user-456" {
					t.Errorf("expected assignee id 'user-456', got %v", people[0]["id"])
				}
			},
		},
		{
			name: "flags override existing properties",
			sf:   nil,
			properties: map[string]interface{}{
				"Status": map[string]interface{}{
					"status": map[string]interface{}{
						"name": "Todo",
					},
				},
			},
			status:   "Done",
			priority: "",
			assignee: "",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				status := result["Status"].(map[string]interface{})
				statusData := status["status"].(map[string]interface{})
				if statusData["name"] != "Done" {
					t.Errorf("expected Status name 'Done' (flag takes precedence), got %v", statusData["name"])
				}
			},
		},
		{
			name: "preserves other properties when adding via flags",
			sf:   nil,
			properties: map[string]interface{}{
				"Title": map[string]interface{}{
					"title": []interface{}{
						map[string]interface{}{"text": map[string]interface{}{"content": "My Page"}},
					},
				},
			},
			status:   "In Progress",
			priority: "",
			assignee: "",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				// Check that Title is preserved
				title, ok := result["Title"].(map[string]interface{})
				if !ok {
					t.Fatal("Title property should be preserved")
				}
				titleArr := title["title"].([]interface{})
				if len(titleArr) != 1 {
					t.Errorf("expected Title to have 1 element")
				}

				// Check that Status was added
				status := result["Status"].(map[string]interface{})
				statusData := status["status"].(map[string]interface{})
				if statusData["name"] != "In Progress" {
					t.Errorf("expected Status name 'In Progress', got %v", statusData["name"])
				}
			},
		},
		{
			name:       "empty flags don't set properties",
			sf:         nil,
			properties: make(map[string]interface{}),
			status:     "",
			priority:   "",
			assignee:   "",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				if len(result) != 0 {
					t.Errorf("expected empty result when no flags set, got %d properties", len(result))
				}
			},
		},
		{
			name: "skill file resolves user alias",
			sf: &skill.SkillFile{
				Users: map[string]skill.UserAlias{
					"alice": {Alias: "alice", ID: "resolved-alice-id"},
				},
			},
			properties: make(map[string]interface{}),
			status:     "",
			priority:   "",
			assignee:   "alice",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				assignee := result["Assignee"].(map[string]interface{})
				people := assignee["people"].([]map[string]interface{})
				if people[0]["id"] != "resolved-alice-id" {
					t.Errorf("expected resolved id 'resolved-alice-id', got %v", people[0]["id"])
				}
			},
		},
		{
			name: "unresolved user alias passes through",
			sf: &skill.SkillFile{
				Users: map[string]skill.UserAlias{
					"bob": {Alias: "bob", ID: "bob-id"},
				},
			},
			properties: make(map[string]interface{}),
			status:     "",
			priority:   "",
			assignee:   "unknown-user",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				assignee := result["Assignee"].(map[string]interface{})
				people := assignee["people"].([]map[string]interface{})
				if people[0]["id"] != "unknown-user" {
					t.Errorf("expected unresolved 'unknown-user', got %v", people[0]["id"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPropertiesFromFlags(tt.sf, tt.properties, tt.status, tt.priority, tt.assignee)
			tt.checkFunc(t, result)
		})
	}
}

func TestBuildPropertiesFromFlags_JSONOutput(t *testing.T) {
	// Verify the output structure matches what the Notion API expects
	result := buildPropertiesFromFlags(nil, nil, "Done", "High", "user-123")

	// Should be serializable to JSON
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result to JSON: %v", err)
	}

	// Verify the JSON structure
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Check Status structure: {"Status":{"status":{"name":"Done"}}}
	status := parsed["Status"].(map[string]interface{})
	statusData := status["status"].(map[string]interface{})
	if statusData["name"] != "Done" {
		t.Errorf("expected Status name 'Done', got %v", statusData["name"])
	}

	// Check Priority structure: {"Priority":{"select":{"name":"High"}}}
	priority := parsed["Priority"].(map[string]interface{})
	selectData := priority["select"].(map[string]interface{})
	if selectData["name"] != "High" {
		t.Errorf("expected Priority name 'High', got %v", selectData["name"])
	}

	// Check Assignee structure: {"Assignee":{"people":[{"object":"user","id":"user-123"}]}}
	assignee := parsed["Assignee"].(map[string]interface{})
	people := assignee["people"].([]interface{})
	person := people[0].(map[string]interface{})
	if person["object"] != "user" {
		t.Errorf("expected object 'user', got %v", person["object"])
	}
	if person["id"] != "user-123" {
		t.Errorf("expected id 'user-123', got %v", person["id"])
	}
}

func TestFindTitlePropertyName(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]map[string]interface{}
		want       string
	}{
		{
			name: "finds title property named Name",
			properties: map[string]map[string]interface{}{
				"Name":   {"type": "title"},
				"Status": {"type": "status"},
			},
			want: "Name",
		},
		{
			name: "finds title property named Title",
			properties: map[string]map[string]interface{}{
				"Title": {"type": "title"},
				"Notes": {"type": "rich_text"},
			},
			want: "Title",
		},
		{
			name: "finds title property with custom name",
			properties: map[string]map[string]interface{}{
				"Task Name": {"type": "title"},
				"Priority":  {"type": "select"},
			},
			want: "Task Name",
		},
		{
			name: "returns default when no title property",
			properties: map[string]map[string]interface{}{
				"Status": {"type": "status"},
				"Notes":  {"type": "rich_text"},
			},
			want: "title",
		},
		{
			name:       "returns default for empty properties",
			properties: map[string]map[string]interface{}{},
			want:       "title",
		},
		{
			name:       "returns default for nil properties",
			properties: nil,
			want:       "title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findTitlePropertyName(tt.properties)
			if got != tt.want {
				t.Errorf("findTitlePropertyName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindTitlePropertyNameFromPage(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]interface{}
		want       string
	}{
		{
			name: "finds title property from page properties",
			properties: map[string]interface{}{
				"Name": map[string]interface{}{
					"type": "title",
					"title": []interface{}{
						map[string]interface{}{"text": map[string]interface{}{"content": "Test"}},
					},
				},
				"Status": map[string]interface{}{
					"type":   "status",
					"status": map[string]interface{}{"name": "Done"},
				},
			},
			want: "Name",
		},
		{
			name: "finds title with different property name",
			properties: map[string]interface{}{
				"Task": map[string]interface{}{
					"type":  "title",
					"title": []interface{}{},
				},
			},
			want: "Task",
		},
		{
			name: "returns default when no title property",
			properties: map[string]interface{}{
				"Status": map[string]interface{}{
					"type": "status",
				},
			},
			want: "title",
		},
		{
			name:       "returns default for empty properties",
			properties: map[string]interface{}{},
			want:       "title",
		},
		{
			name:       "returns default for nil properties",
			properties: nil,
			want:       "title",
		},
		{
			name: "skips non-map property values",
			properties: map[string]interface{}{
				"BadProp": "not a map",
				"Name": map[string]interface{}{
					"type": "title",
				},
			},
			want: "Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findTitlePropertyNameFromPage(tt.properties)
			if got != tt.want {
				t.Errorf("findTitlePropertyNameFromPage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetTitleProperty(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]interface{}
		propName   string
		title      string
		checkFunc  func(t *testing.T, result map[string]interface{})
	}{
		{
			name:       "sets title property with Name",
			properties: make(map[string]interface{}),
			propName:   "Name",
			title:      "My Page Title",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				nameProp, ok := result["Name"].(map[string]interface{})
				if !ok {
					t.Fatal("Name property should be a map")
				}
				titleArr, ok := nameProp["title"].([]map[string]interface{})
				if !ok {
					t.Fatal("title should be an array of maps")
				}
				if len(titleArr) != 1 {
					t.Fatalf("expected 1 element in title array, got %d", len(titleArr))
				}
				textContent := titleArr[0]["text"].(map[string]interface{})
				if textContent["content"] != "My Page Title" {
					t.Errorf("expected content 'My Page Title', got %v", textContent["content"])
				}
			},
		},
		{
			name:       "sets title property with custom name",
			properties: make(map[string]interface{}),
			propName:   "Task Name",
			title:      "Task Title",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				taskProp, ok := result["Task Name"].(map[string]interface{})
				if !ok {
					t.Fatal("Task Name property should be a map")
				}
				titleArr, ok := taskProp["title"].([]map[string]interface{})
				if !ok {
					t.Fatal("title should be an array of maps")
				}
				textContent := titleArr[0]["text"].(map[string]interface{})
				if textContent["content"] != "Task Title" {
					t.Errorf("expected content 'Task Title', got %v", textContent["content"])
				}
			},
		},
		{
			name:       "creates properties map when nil",
			properties: nil,
			propName:   "Title",
			title:      "New Title",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				if result == nil {
					t.Fatal("expected non-nil result")
				}
				titleProp, ok := result["Title"].(map[string]interface{})
				if !ok {
					t.Fatal("Title property should be a map")
				}
				titleArr := titleProp["title"].([]map[string]interface{})
				textContent := titleArr[0]["text"].(map[string]interface{})
				if textContent["content"] != "New Title" {
					t.Errorf("expected content 'New Title', got %v", textContent["content"])
				}
			},
		},
		{
			name: "preserves existing properties",
			properties: map[string]interface{}{
				"Status": map[string]interface{}{
					"status": map[string]interface{}{"name": "Done"},
				},
			},
			propName: "Name",
			title:    "Test",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				// Check Status is preserved
				status, ok := result["Status"].(map[string]interface{})
				if !ok {
					t.Fatal("Status should be preserved")
				}
				statusData := status["status"].(map[string]interface{})
				if statusData["name"] != "Done" {
					t.Error("Status value should be preserved")
				}
				// Check Name was added
				nameProp, ok := result["Name"].(map[string]interface{})
				if !ok {
					t.Fatal("Name property should be added")
				}
				titleArr := nameProp["title"].([]map[string]interface{})
				textContent := titleArr[0]["text"].(map[string]interface{})
				if textContent["content"] != "Test" {
					t.Errorf("expected content 'Test', got %v", textContent["content"])
				}
			},
		},
		{
			name: "overrides existing title property",
			properties: map[string]interface{}{
				"Name": map[string]interface{}{
					"title": []map[string]interface{}{
						{"text": map[string]interface{}{"content": "Old Title"}},
					},
				},
			},
			propName: "Name",
			title:    "New Title",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				nameProp := result["Name"].(map[string]interface{})
				titleArr := nameProp["title"].([]map[string]interface{})
				textContent := titleArr[0]["text"].(map[string]interface{})
				if textContent["content"] != "New Title" {
					t.Errorf("expected content 'New Title', got %v", textContent["content"])
				}
			},
		},
		{
			name:       "handles empty title",
			properties: make(map[string]interface{}),
			propName:   "Name",
			title:      "",
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				nameProp := result["Name"].(map[string]interface{})
				titleArr := nameProp["title"].([]map[string]interface{})
				textContent := titleArr[0]["text"].(map[string]interface{})
				if textContent["content"] != "" {
					t.Errorf("expected empty content, got %v", textContent["content"])
				}
			},
		},
		{
			name:       "handles title with special characters",
			properties: make(map[string]interface{}),
			propName:   "Name",
			title:      `Title with "quotes" and 'apostrophes' & <special> chars`,
			checkFunc: func(t *testing.T, result map[string]interface{}) {
				nameProp := result["Name"].(map[string]interface{})
				titleArr := nameProp["title"].([]map[string]interface{})
				textContent := titleArr[0]["text"].(map[string]interface{})
				expected := `Title with "quotes" and 'apostrophes' & <special> chars`
				if textContent["content"] != expected {
					t.Errorf("expected content %q, got %v", expected, textContent["content"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := setTitleProperty(tt.properties, tt.propName, tt.title)
			tt.checkFunc(t, result)
		})
	}
}

func TestSetTitleProperty_JSONOutput(t *testing.T) {
	// Verify the output structure matches what the Notion API expects
	result := setTitleProperty(nil, "Name", "Test Page")

	// Should be serializable to JSON
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result to JSON: %v", err)
	}

	// Verify the JSON structure matches Notion API format
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Check structure: {"Name":{"title":[{"text":{"content":"Test Page"}}]}}
	nameProp := parsed["Name"].(map[string]interface{})
	titleArr := nameProp["title"].([]interface{})
	if len(titleArr) != 1 {
		t.Fatalf("expected 1 element in title array, got %d", len(titleArr))
	}
	firstElem := titleArr[0].(map[string]interface{})
	textContent := firstElem["text"].(map[string]interface{})
	if textContent["content"] != "Test Page" {
		t.Errorf("expected content 'Test Page', got %v", textContent["content"])
	}
}
