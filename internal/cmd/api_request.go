package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/notion-cli/internal/cmdutil"
	"github.com/salmonumbrella/notion-cli/internal/errors"
	"github.com/salmonumbrella/notion-cli/internal/notion"
)

func newAPIRequestCmd() *cobra.Command {
	var bodyJSON string
	var bodyFile string
	var paginate bool
	var raw bool
	var includeHeaders bool
	var headers []string
	var noAuth bool

	cmd := &cobra.Command{
		Use:   "request <method> <path>",
		Short: "Make a raw Notion API request",
		Long: `Make a raw Notion API request (useful for new endpoints and debugging).

Examples:
  ntn api request GET /users
  ntn api request POST /search --body '{"query":"project"}'
  ntn api request POST /databases/<id>/query --body @query.json
  ntn api request GET /blocks/<id>/children --paginate
  ntn api request GET /users --no-auth
  ntn api request GET /users --header "Authorization: Bearer <token>"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(strings.TrimSpace(args[0]))
			path := strings.TrimSpace(args[1])

			bodyStr, err := cmdutil.ResolveJSONInput(bodyJSON, bodyFile)
			if err != nil {
				return err
			}

			bodyStr = cmdutil.NormalizeJSONInput(bodyStr)
			var bodyBytes []byte
			if bodyStr != "" {
				if !json.Valid([]byte(bodyStr)) {
					return fmt.Errorf("invalid JSON body")
				}
				bodyBytes = []byte(bodyStr)
			}

			customHeaders, err := parseHeaderFlags(headers)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			disableAuth := noAuth || hasAuthorizationHeader(customHeaders)
			var token string
			if !disableAuth {
				token, err = GetTokenFromContext(ctx)
				if err != nil {
					return errors.AuthRequiredError(err)
				}
			}

			client := NewNotionClient(ctx, token)
			if disableAuth {
				client.WithAuthHeaderDisabled()
			}

			if paginate {
				return runPaginatedAPIRequest(ctx, client, method, path, bodyBytes, customHeaders, raw, includeHeaders)
			}

			resp, err := client.DoRawRequest(ctx, method, path, bodyBytes, customHeaders)
			if err != nil {
				return err
			}

			return renderAPIResponse(ctx, resp, raw, includeHeaders)
		},
	}

	cmd.Flags().StringVar(&bodyJSON, "body", "", "JSON request body (or @file, @-, or - for stdin)")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read JSON body from file ('-' for stdin)")
	cmd.Flags().BoolVar(&paginate, "paginate", false, "Automatically paginate results when supported")
	cmd.Flags().BoolVar(&raw, "raw", false, "Output only the response body")
	cmd.Flags().BoolVar(&includeHeaders, "include-headers", false, "Include response headers in output")
	cmd.Flags().StringArrayVar(&headers, "header", nil, "Custom header (repeatable, format: 'Key: Value')")
	cmd.Flags().BoolVar(&noAuth, "no-auth", false, "Disable default Authorization header (also disabled when Authorization header is provided)")

	_ = cmd.Flags().MarkHidden("body-file") // prefer @file style, keep as fallback

	return cmd
}

func runPaginatedAPIRequest(ctx context.Context, client rawRequester, method, path string, body []byte, headers http.Header, raw bool, includeHeaders bool) error {
	var allResults []interface{}
	var lastResponse *notion.RawResponse
	var nextCursor string

	for {
		reqPath := path
		reqBody := body

		if method == http.MethodGet {
			reqPath = addQueryParam(reqPath, "start_cursor", nextCursor)
		} else if len(reqBody) > 0 || method == http.MethodPost || method == http.MethodPatch {
			updatedBody, err := withStartCursor(reqBody, nextCursor)
			if err != nil {
				return err
			}
			reqBody = updatedBody
		}

		resp, err := client.DoRawRequest(ctx, method, reqPath, reqBody, headers)
		if err != nil {
			return err
		}
		lastResponse = resp

		payload, ok := decodeJSONBody(resp.Body)
		if !ok {
			break
		}

		results, hasMore, cursor, ok := extractPagination(payload)
		if !ok {
			break
		}

		allResults = append(allResults, results...)
		if !hasMore || cursor == "" {
			break
		}
		nextCursor = cursor
	}

	if lastResponse == nil {
		return fmt.Errorf("no response returned")
	}

	if raw {
		if len(allResults) > 0 {
			printer := printerForContext(ctx)
			return printer.Print(ctx, allResults)
		}
		return renderAPIResponse(ctx, lastResponse, raw, includeHeaders)
	}

	if len(allResults) > 0 {
		envelope := buildAPIEnvelope(lastResponse, map[string]interface{}{
			"results":  allResults,
			"has_more": false,
		}, includeHeaders)
		printer := printerForContext(ctx)
		return printer.Print(ctx, envelope)
	}

	return renderAPIResponse(ctx, lastResponse, raw, includeHeaders)
}

func renderAPIResponse(ctx context.Context, resp *notion.RawResponse, raw bool, includeHeaders bool) error {
	if resp == nil {
		return fmt.Errorf("no response returned")
	}

	bodyPayload, isJSON := decodeJSONBody(resp.Body)
	if raw {
		if isJSON {
			printer := printerForContext(ctx)
			return printer.Print(ctx, bodyPayload)
		}
		_, _ = fmt.Fprintln(stdoutFromContext(ctx), string(resp.Body))
		return nil
	}

	envelope := buildAPIEnvelope(resp, bodyPayload, includeHeaders)
	printer := printerForContext(ctx)
	return printer.Print(ctx, envelope)
}

func buildAPIEnvelope(resp *notion.RawResponse, body interface{}, includeHeaders bool) map[string]interface{} {
	envelope := map[string]interface{}{
		"status":     resp.StatusCode,
		"request_id": resp.Headers.Get("X-Request-Id"),
		"body":       body,
	}

	if includeHeaders {
		envelope["headers"] = flattenHeaders(resp.Headers)
	}

	return envelope
}

func decodeJSONBody(body []byte) (interface{}, bool) {
	if len(body) == 0 {
		return nil, true
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, true
	}

	if !json.Valid(trimmed) {
		return string(body), false
	}

	var payload interface{}
	if err := json.Unmarshal(trimmed, &payload); err != nil {
		return string(body), false
	}

	return payload, true
}

func extractPagination(payload interface{}) ([]interface{}, bool, string, bool) {
	respMap, ok := payload.(map[string]interface{})
	if !ok {
		return nil, false, "", false
	}
	resultsRaw, ok := respMap["results"].([]interface{})
	if !ok {
		return nil, false, "", false
	}
	hasMore, _ := respMap["has_more"].(bool)
	cursor, _ := respMap["next_cursor"].(string)
	return resultsRaw, hasMore, cursor, true
}

func withStartCursor(body []byte, cursor string) ([]byte, error) {
	if cursor == "" {
		return body, nil
	}

	var payload map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("invalid JSON body for pagination: %w", err)
		}
	} else {
		payload = map[string]interface{}{}
	}

	payload["start_cursor"] = cursor
	return json.Marshal(payload)
}

func addQueryParam(path string, key string, value string) string {
	if value == "" {
		return path
	}

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		u, err := url.Parse(path)
		if err != nil {
			return path
		}
		q := u.Query()
		q.Set(key, value)
		u.RawQuery = q.Encode()
		return u.String()
	}

	parts := strings.SplitN(path, "?", 2)
	base := parts[0]
	q := url.Values{}
	if len(parts) == 2 {
		if parsed, err := url.ParseQuery(parts[1]); err == nil {
			q = parsed
		}
	}
	q.Set(key, value)
	if strings.HasPrefix(base, "/") {
		return base + "?" + q.Encode()
	}
	return "/" + base + "?" + q.Encode()
}

func parseHeaderFlags(values []string) (http.Header, error) {
	headers := http.Header{}
	for _, raw := range values {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header %q (expected 'Key: Value')", raw)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid header %q (missing key)", raw)
		}
		headers.Add(key, val)
	}
	return headers, nil
}

func hasAuthorizationHeader(headers http.Header) bool {
	if headers == nil {
		return false
	}
	_, ok := headers[http.CanonicalHeaderKey("Authorization")]
	return ok
}

func flattenHeaders(headers http.Header) map[string]string {
	flat := map[string]string{}
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		flat[key] = strings.Join(values, ", ")
	}
	return flat
}
