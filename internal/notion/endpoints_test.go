// internal/notion/endpoints_test.go
package notion

import "testing"

func TestEndpoints_Path(t *testing.T) {
	tests := []struct {
		endpoint Endpoint
		expected string
	}{
		{Endpoints.Pages.Get, "/pages/{id}"},
		{Endpoints.Pages.Create, "/pages"},
		{Endpoints.Blocks.Children, "/blocks/{id}/children"},
		{Endpoints.Search, "/search"},
	}

	for _, tt := range tests {
		if tt.endpoint.Path != tt.expected {
			t.Errorf("expected path %q, got %q", tt.expected, tt.endpoint.Path)
		}
	}
}

func TestEndpoints_IsIdempotent(t *testing.T) {
	if !Endpoints.Pages.Get.Idempotent {
		t.Error("GET pages should be idempotent")
	}
	if Endpoints.Pages.Create.Idempotent {
		t.Error("POST pages should not be idempotent")
	}
}

func TestEndpoints_Method(t *testing.T) {
	if Endpoints.Pages.Get.Method != "GET" {
		t.Errorf("expected GET, got %s", Endpoints.Pages.Get.Method)
	}
	if Endpoints.Pages.Create.Method != "POST" {
		t.Errorf("expected POST, got %s", Endpoints.Pages.Create.Method)
	}
	if Endpoints.Pages.Update.Method != "PATCH" {
		t.Errorf("expected PATCH, got %s", Endpoints.Pages.Update.Method)
	}
}
