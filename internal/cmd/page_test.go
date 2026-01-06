package cmd

import (
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
