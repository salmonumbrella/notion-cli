package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	ctxerrors "github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
	"github.com/salmonumbrella/notion-cli/internal/output"
)

func validateErrorFormat(format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "auto", "text", "json", "yaml":
		return nil
	default:
		return fmt.Errorf("invalid --error-format %q (expected auto|text|json|yaml)", format)
	}
}

func effectiveErrorFormat() string {
	format := strings.ToLower(strings.TrimSpace(errorFormat))
	if format == "" || format == "auto" {
		switch outputFormat {
		case output.FormatJSON, output.FormatNDJSON:
			return "json"
		case output.FormatYAML:
			return "yaml"
		default:
			return "text"
		}
	}
	return format
}

func printCommandError(err error) {
	if err == nil {
		return
	}

	switch effectiveErrorFormat() {
	case "json":
		enc := json.NewEncoder(os.Stderr)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(buildErrorEnvelope(err))
		return
	case "yaml":
		enc := yaml.NewEncoder(os.Stderr)
		enc.SetIndent(2)
		_ = enc.Encode(buildErrorEnvelope(err))
		_ = enc.Close()
		return
	}

	fmt.Fprintln(os.Stderr, err)
}

func buildErrorEnvelope(err error) map[string]interface{} {
	payload := map[string]interface{}{
		"error": map[string]interface{}{
			"message": err.Error(),
		},
	}

	var contextual *ctxerrors.ContextualError
	if errors.As(err, &contextual) {
		errMap := payload["error"].(map[string]interface{})
		errMap["method"] = contextual.Method
		errMap["url"] = contextual.URL
		if contextual.StatusCode > 0 {
			errMap["status"] = contextual.StatusCode
		}
	}

	var apiErr *notion.APIError
	if errors.As(err, &apiErr) {
		errMap := payload["error"].(map[string]interface{})
		errMap["type"] = "notion_api"
		if apiErr.StatusCode > 0 {
			errMap["status"] = apiErr.StatusCode
		}
		if apiErr.Response != nil {
			errMap["code"] = apiErr.Response.Code
			errMap["message"] = apiErr.Response.Message
			if apiErr.Response.Status > 0 {
				errMap["status"] = apiErr.Response.Status
			}
		}
		if apiErr.RetryAfter > 0 {
			errMap["retry_after_seconds"] = int(apiErr.RetryAfter.Seconds())
		}
	}

	var rlErr *ctxerrors.RateLimitError
	if errors.As(err, &rlErr) {
		errMap := payload["error"].(map[string]interface{})
		errMap["type"] = "rate_limit"
		errMap["retry_after_seconds"] = int(rlErr.RetryAfter.Seconds())
	}

	var authErr *ctxerrors.AuthError
	if errors.As(err, &authErr) {
		errMap := payload["error"].(map[string]interface{})
		errMap["type"] = "auth"
	}

	var validationErr *ctxerrors.ValidationError
	if errors.As(err, &validationErr) {
		errMap := payload["error"].(map[string]interface{})
		errMap["type"] = "validation"
		errMap["field"] = validationErr.Field
	}

	if ctxerrors.IsCircuitBreakerError(err) {
		errMap := payload["error"].(map[string]interface{})
		errMap["type"] = "circuit_breaker"
	}

	return payload
}
