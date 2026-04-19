package mcp

import (
	"context"
	"time"
)

// MCPClient is the subset of client behavior consumed by the CLI handlers.
// It exists so handlers can be tested with a FakeClient without spinning up
// an httptest.Server. The concrete implementation is *Client.
type MCPClient interface {
	Initialize(ctx context.Context) (*InitializeResult, time.Duration, error)
	ListTools(ctx context.Context) ([]ToolDefinition, error)
	CallTool(ctx context.Context, name string, args map[string]any) (*RPCResponse, error)
	ListResources(ctx context.Context) ([]Resource, error)
	ReadResource(ctx context.Context, uri string) ([]ResourceContent, error)
}
