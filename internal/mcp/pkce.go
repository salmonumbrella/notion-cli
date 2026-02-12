package mcp

import (
	mcpclient "github.com/mark3labs/mcp-go/client"
)

// Re-export mcp-go PKCE utilities so they can be used within this package
// (including tests which share the same package name).

var (
	mcp_generateCodeVerifier  = mcpclient.GenerateCodeVerifier
	mcp_generateCodeChallenge = mcpclient.GenerateCodeChallenge
	mcp_generateState         = mcpclient.GenerateState
)
