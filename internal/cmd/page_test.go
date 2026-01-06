package cmd

import (
	"encoding/json"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformPropertiesWithMentions(tt.properties, tt.userIDs)
			tt.checkFunc(t, result)
		})
	}
}

func TestTransformPropertiesWithMentions_JSONOutput(t *testing.T) {
	// Test that the output can be serialized to valid JSON
	properties := map[string]interface{}{
		"Summary": "@Georges should review",
	}
	userIDs := []string{"georges-user-id"}

	result := transformPropertiesWithMentions(properties, userIDs)

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
