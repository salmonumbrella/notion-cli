package cmd

import (
	"errors"
	"testing"

	ctxerrors "github.com/salmonumbrella/notion-cli/internal/errors"
)

func TestBuildErrorEnvelope_UserError(t *testing.T) {
	err := ctxerrors.NewUserError("invalid flag", "Use --help to see valid flags")
	env := buildErrorEnvelope(err)

	payload, ok := env["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error map, got %T", env["error"])
	}

	if payload["category"] != "user" {
		t.Errorf("category = %v, want user", payload["category"])
	}
	if payload["suggestion"] != "Use --help to see valid flags" {
		t.Errorf("suggestion = %v, want %q", payload["suggestion"], "Use --help to see valid flags")
	}
}

func TestBuildErrorEnvelope_ValidationError(t *testing.T) {
	err := &ctxerrors.ValidationError{Field: "name", Message: "required"}
	env := buildErrorEnvelope(err)

	payload, ok := env["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error map, got %T", env["error"])
	}

	if payload["category"] != "user" {
		t.Errorf("category = %v, want user", payload["category"])
	}
	if payload["type"] != "validation" {
		t.Errorf("type = %v, want validation", payload["type"])
	}
}

func TestBuildErrorEnvelope_SystemError(t *testing.T) {
	err := errors.New("boom")
	env := buildErrorEnvelope(err)

	payload, ok := env["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error map, got %T", env["error"])
	}

	if payload["category"] != "system" {
		t.Errorf("category = %v, want system", payload["category"])
	}
	if _, ok := payload["suggestion"]; ok {
		t.Errorf("expected no suggestion for system error")
	}
}
