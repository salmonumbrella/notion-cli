package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		return ctxerrors.NewUserError(
			fmt.Sprintf("invalid --error-format %q", format),
			"Use one of: auto, text, json, yaml",
		)
	}
}

func effectiveErrorFormat(ctx context.Context) string {
	format := strings.ToLower(strings.TrimSpace(ErrorFormatFromContext(ctx)))
	if format == "" || format == "auto" {
		switch output.FormatFromContext(ctx) {
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

func printCommandError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	switch effectiveErrorFormat(ctx) {
	case "json":
		enc := json.NewEncoder(stderrFromContext(ctx))
		enc.SetEscapeHTML(false)
		_ = enc.Encode(buildErrorEnvelope(err))
		return
	case "yaml":
		enc := yaml.NewEncoder(stderrFromContext(ctx))
		enc.SetIndent(2)
		_ = enc.Encode(buildErrorEnvelope(err))
		_ = enc.Close()
		return
	}

	_, _ = fmt.Fprintln(stderrFromContext(ctx), err)
	if suggestion := ctxerrors.UserSuggestion(err); suggestion != "" {
		_, _ = fmt.Fprintf(stderrFromContext(ctx), "Hint: %s\n", suggestion)
	}
}

func buildErrorEnvelope(err error) map[string]interface{} {
	payload := map[string]interface{}{
		"error": map[string]interface{}{
			"message": err.Error(),
		},
	}

	errMap := payload["error"].(map[string]interface{})
	category := "system"
	if ctxerrors.IsUserError(err) || ctxerrors.IsValidationError(err) || ctxerrors.IsAuthError(err) {
		category = "user"
	}
	errMap["category"] = category

	if suggestion := ctxerrors.UserSuggestion(err); suggestion != "" {
		errMap["suggestion"] = suggestion
	}

	var contextual *ctxerrors.ContextualError
	if errors.As(err, &contextual) {
		errMap["method"] = contextual.Method
		errMap["url"] = contextual.URL
		if contextual.StatusCode > 0 {
			errMap["status"] = contextual.StatusCode
		}
	}

	var apiErr *notion.APIError
	if errors.As(err, &apiErr) {
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
		errMap["type"] = "rate_limit"
		errMap["retry_after_seconds"] = int(rlErr.RetryAfter.Seconds())
	}

	var authErr *ctxerrors.AuthError
	if errors.As(err, &authErr) {
		errMap["type"] = "auth"
	}

	var validationErr *ctxerrors.ValidationError
	if errors.As(err, &validationErr) {
		errMap["type"] = "validation"
		errMap["field"] = validationErr.Field
	}

	if ctxerrors.IsCircuitBreakerError(err) {
		errMap["type"] = "circuit_breaker"
	}

	return payload
}
