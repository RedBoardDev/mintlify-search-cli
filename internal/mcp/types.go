package mcp

import "encoding/json"

type Discovery struct {
	Server       ServerInfo          `json:"server"`
	Capabilities DiscoveryCapability `json:"capabilities"`
}

type ServerInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Transport string `json:"transport"`
}

type DiscoveryCapability struct {
	Tools map[string]ToolDefinition `json:"tools"`
}

type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string                    `json:"type"`
	Required   []string                  `json:"required"`
	Properties map[string]SchemaProperty `json:"properties"`
}

type SchemaProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    InitializeFeatures `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type InitializeFeatures struct {
	Tools map[string]json.RawMessage `json:"tools"`
}

type ToolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}
