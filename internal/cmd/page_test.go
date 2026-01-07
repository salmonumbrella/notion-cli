package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
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
				"Summary": "@Georges should film this",
			},
			userIDs: []string{"georges-user-id"},
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
				if user["id"] != "georges-user-id" {
					t.Errorf("expected user id georges-user-id, got %v", user["id"])
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
		"Summary": "@Georges should **review** this",
		"Notes":   "Plain text",
	}
	userIDs := []string{"georges-user-id"}

	resultVerbose, usedVerbose := transformPropertiesWithMentionsVerbose(io.Discard, properties, userIDs, true)
	resultNonVerbose, usedNonVerbose := transformPropertiesWithMentionsVerbose(io.Discard, properties, userIDs, false)

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
		"Summary": "@Georges should **review** this",
	}
	userIDs := []string{"georges-user-id"}

	var buf bytes.Buffer
	_, _ = transformPropertiesWithMentionsVerbose(&buf, properties, userIDs, true)

	output := buf.String()
	if !strings.Contains(output, `Property "Summary"`) {
		t.Errorf("verbose output should contain property name, got: %s", output)
	}
	// Check that mention assignments are shown
	if !strings.Contains(output, "Mentions:") {
		t.Errorf("verbose output should contain 'Mentions:' header, got: %s", output)
	}
	if !strings.Contains(output, "@Georges → georges-user-id") {
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
	_, _ = transformPropertiesWithMentionsVerbose(&buf, properties, userIDs, true)

	output := buf.String()
	if !strings.Contains(output, "@Alice → alice-id") {
		t.Errorf("verbose output should show @Alice mapped to alice-id, got: %s", output)
	}
	if !strings.Contains(output, "@Bob → (no user ID available)") {
		t.Errorf("verbose output should show @Bob with no user ID, got: %s", output)
	}
}

func TestTransformPropertiesWithMentions_JSONOutput(t *testing.T) {
	// Test that the output can be serialized to valid JSON
	properties := map[string]interface{}{
		"Summary": "@Georges should review",
	}
	userIDs := []string{"georges-user-id"}

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
