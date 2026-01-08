package cmd

import (
	"errors"
	"testing"
)

func TestIsArchivedBlockError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "archived block error",
			err:      errors.New("notion API error 400 (validation_error): Can't edit block that is archived. You must unarchive the block before editing."),
			expected: true,
		},
		{
			name:     "different error",
			err:      errors.New("notion API error 404: block not found"),
			expected: false,
		},
		{
			name:     "partial match - archived only",
			err:      errors.New("block is archived"),
			expected: false,
		},
		{
			name:     "partial match - edit only",
			err:      errors.New("cannot edit block"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isArchivedBlockError(tt.err)
			if result != tt.expected {
				t.Errorf("isArchivedBlockError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestPtrBool(t *testing.T) {
	truePtr := ptrBool(true)
	if truePtr == nil || *truePtr != true {
		t.Errorf("ptrBool(true) = %v, want pointer to true", truePtr)
	}

	falsePtr := ptrBool(false)
	if falsePtr == nil || *falsePtr != false {
		t.Errorf("ptrBool(false) = %v, want pointer to false", falsePtr)
	}
}
