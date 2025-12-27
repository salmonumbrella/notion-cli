package output

import (
	"context"
	"testing"
)

func TestWithFormat(t *testing.T) {
	tests := []struct {
		name   string
		format Format
	}{
		{name: "text format", format: FormatText},
		{name: "json format", format: FormatJSON},
		{name: "table format", format: FormatTable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = WithFormat(ctx, tt.format)

			got := FormatFromContext(ctx)
			if got != tt.format {
				t.Errorf("FormatFromContext() = %v, want %v", got, tt.format)
			}
		})
	}
}

func TestFormatFromContext_Default(t *testing.T) {
	// Empty context should return default format (FormatText)
	ctx := context.Background()
	got := FormatFromContext(ctx)
	want := FormatText

	if got != want {
		t.Errorf("FormatFromContext() with empty context = %v, want %v", got, want)
	}
}

func TestFormatFromContext_WrongType(t *testing.T) {
	// Context with wrong type should return default format
	ctx := context.WithValue(context.Background(), contextKey{}, "not-a-format")
	got := FormatFromContext(ctx)
	want := FormatText

	if got != want {
		t.Errorf("FormatFromContext() with wrong type = %v, want %v", got, want)
	}
}

func TestFormatFromContext_Nested(t *testing.T) {
	// Test that nested contexts preserve the format
	ctx := context.Background()
	ctx = WithFormat(ctx, FormatJSON)

	// Create a child context with a different value
	type otherKey struct{}
	ctx = context.WithValue(ctx, otherKey{}, "some-value")

	// Format should still be preserved
	got := FormatFromContext(ctx)
	want := FormatJSON

	if got != want {
		t.Errorf("FormatFromContext() in nested context = %v, want %v", got, want)
	}
}

func TestFormatFromContext_Override(t *testing.T) {
	// Test that format can be overridden in a child context
	ctx := context.Background()
	ctx = WithFormat(ctx, FormatText)

	// Override with a new format
	ctx = WithFormat(ctx, FormatJSON)

	got := FormatFromContext(ctx)
	want := FormatJSON

	if got != want {
		t.Errorf("FormatFromContext() after override = %v, want %v", got, want)
	}
}

func TestLightFromContext_Default(t *testing.T) {
	if LightFromContext(context.Background()) {
		t.Fatal("LightFromContext() should default to false")
	}
}

func TestWithLight_RoundTrip(t *testing.T) {
	ctx := WithLight(context.Background(), true)
	if !LightFromContext(ctx) {
		t.Fatal("LightFromContext() should return true after WithLight")
	}
}

func TestCompactJSONFromContext_Default(t *testing.T) {
	if CompactJSONFromContext(context.Background()) {
		t.Fatal("CompactJSONFromContext() should default to false")
	}
}

func TestWithCompactJSON_RoundTrip(t *testing.T) {
	ctx := WithCompactJSON(context.Background(), true)
	if !CompactJSONFromContext(ctx) {
		t.Fatal("CompactJSONFromContext() should return true after WithCompactJSON")
	}
}
