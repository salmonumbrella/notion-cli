package cmd

import "testing"

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
