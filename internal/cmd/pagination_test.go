package cmd

import (
	"context"
	"testing"

	"github.com/salmonumbrella/notion-cli/internal/output"
)

func TestCapPageSize(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int
		limit    int
		want     int
	}{
		{
			name:     "no limit returns pageSize unchanged",
			pageSize: 50,
			limit:    0,
			want:     50,
		},
		{
			name:     "no limit with zero pageSize returns zero",
			pageSize: 0,
			limit:    0,
			want:     0,
		},
		{
			name:     "limit smaller than pageSize caps to limit",
			pageSize: 100,
			limit:    25,
			want:     25,
		},
		{
			name:     "limit larger than NotionMaxPageSize caps to max",
			pageSize: 0,
			limit:    200,
			want:     NotionMaxPageSize,
		},
		{
			name:     "limit equals NotionMaxPageSize",
			pageSize: 0,
			limit:    100,
			want:     100,
		},
		{
			name:     "pageSize zero with small limit uses limit",
			pageSize: 0,
			limit:    10,
			want:     10,
		},
		{
			name:     "pageSize smaller than limit stays unchanged",
			pageSize: 20,
			limit:    50,
			want:     20,
		},
		{
			name:     "pageSize equals limit stays unchanged",
			pageSize: 30,
			limit:    30,
			want:     30,
		},
		{
			name:     "pageSize larger than limit gets capped",
			pageSize: 80,
			limit:    40,
			want:     40,
		},
		{
			name:     "limit at boundary 100 with larger pageSize",
			pageSize: 150,
			limit:    100,
			want:     100,
		},
		{
			name:     "limit at boundary 101 caps to max",
			pageSize: 0,
			limit:    101,
			want:     NotionMaxPageSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := capPageSize(tt.pageSize, tt.limit)
			if got != tt.want {
				t.Errorf("capPageSize(%d, %d) = %d, want %d",
					tt.pageSize, tt.limit, got, tt.want)
			}
		})
	}
}

func TestNotionMaxPageSize(t *testing.T) {
	if NotionMaxPageSize != 100 {
		t.Errorf("NotionMaxPageSize = %d, want 100", NotionMaxPageSize)
	}
}

func TestCapPageSize_WithContextLimit(t *testing.T) {
	// Integration test: simulates the command flow where limit comes from context
	// and capPageSize is used to determine effective page size.
	tests := []struct {
		name         string
		contextLimit int
		pageSize     int
		wantPageSize int
	}{
		{
			name:         "limit 500 from context caps page size to 100",
			contextLimit: 500,
			pageSize:     0, // unset, should use capped limit
			wantPageSize: NotionMaxPageSize,
		},
		{
			name:         "limit 200 from context caps page size to 100",
			contextLimit: 200,
			pageSize:     0,
			wantPageSize: NotionMaxPageSize,
		},
		{
			name:         "limit 101 from context caps page size to 100",
			contextLimit: 101,
			pageSize:     0,
			wantPageSize: NotionMaxPageSize,
		},
		{
			name:         "limit 100 from context uses full limit",
			contextLimit: 100,
			pageSize:     0,
			wantPageSize: 100,
		},
		{
			name:         "limit 50 from context uses limit as page size",
			contextLimit: 50,
			pageSize:     0,
			wantPageSize: 50,
		},
		{
			name:         "limit 500 with explicit page size 50 uses page size",
			contextLimit: 500,
			pageSize:     50,
			wantPageSize: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate context with limit (as done in commands)
			ctx := output.WithLimit(context.Background(), tt.contextLimit)
			limit := output.LimitFromContext(ctx)

			// This is the integration: limit from context + capPageSize
			effectivePageSize := capPageSize(tt.pageSize, limit)

			if effectivePageSize != tt.wantPageSize {
				t.Errorf("capPageSize(%d, %d) = %d, want %d (contextLimit=%d)",
					tt.pageSize, limit, effectivePageSize, tt.wantPageSize, tt.contextLimit)
			}

			// Verify page size never exceeds NotionMaxPageSize
			if effectivePageSize > NotionMaxPageSize {
				t.Errorf("effectivePageSize %d exceeds NotionMaxPageSize %d",
					effectivePageSize, NotionMaxPageSize)
			}
		})
	}
}
