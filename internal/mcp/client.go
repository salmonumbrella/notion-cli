package mcp

import (
	"context"
	"fmt"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	// NotionMCPURL is the endpoint for Notion's remote MCP server.
	NotionMCPURL = "https://mcp.notion.com/mcp"

	clientName    = "notion-cli"
	clientVersion = "0.1.0"
)

// Client wraps the mcp-go client for Notion's MCP server.
type Client struct {
	inner *mcpclient.Client
}

// NewClient connects to Notion's MCP server using the provided Bearer token.
// The caller must call Close when done.
func NewClient(ctx context.Context, token string) (*Client, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	inner, err := mcpclient.NewStreamableHttpClient(
		NotionMCPURL,
		transport.WithHTTPHeaders(headers),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP transport: %w", err)
	}

	if err := inner.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start MCP client: %w", err)
	}

	// Initialize the MCP session.
	_, err = inner.Initialize(ctx, mcp.InitializeRequest{
		Params: struct {
			ProtocolVersion string                 `json:"protocolVersion"`
			Capabilities    mcp.ClientCapabilities `json:"capabilities"`
			ClientInfo      mcp.Implementation     `json:"clientInfo"`
		}{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    clientName,
				Version: clientVersion,
			},
		},
	})
	if err != nil {
		_ = inner.Close()
		return nil, fmt.Errorf("failed to initialize MCP session: %w", err)
	}

	return &Client{inner: inner}, nil
}

// CallTool invokes an MCP tool by name with the given arguments and returns
// the concatenated text content from the result.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	result, err := c.inner.CallTool(ctx, mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	if err != nil {
		return "", fmt.Errorf("MCP tool %q failed: %w", name, err)
	}

	if result.IsError {
		return "", fmt.Errorf("MCP tool %q returned error: %s", name, extractText(result))
	}

	return extractText(result), nil
}

// ListTools returns the tools advertised by the Notion MCP server.
func (c *Client) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	result, err := c.inner.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP tools: %w", err)
	}
	return result.Tools, nil
}

// Close terminates the MCP connection.
func (c *Client) Close() error {
	return c.inner.Close()
}

// extractText concatenates all text content from a CallToolResult.
func extractText(result *mcp.CallToolResult) string {
	var text string
	for _, c := range result.Content {
		if tc, ok := mcp.AsTextContent(c); ok {
			if text != "" {
				text += "\n"
			}
			text += tc.Text
		}
	}
	return text
}
