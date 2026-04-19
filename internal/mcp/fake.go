package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// FakeClient is an in-memory MCPClient used by handler tests. Populate the
// exported fields with canned responses; the Call* methods will replay them
// and record every invocation in CallLog.
type FakeClient struct {
	InitResult      *InitializeResult
	InitDuration    time.Duration
	InitErr         error
	Tools           []ToolDefinition
	ToolsErr        error
	ToolResponses   map[string]*RPCResponse
	ToolErrors      map[string]error
	Resources       []Resource
	ResourcesErr    error
	ResourceContent map[string][]ResourceContent
	ResourceErr     error
	CallLog         []CallRecord
}

// CallRecord captures a single MCPClient invocation for test assertions.
type CallRecord struct {
	Method string
	Name   string
	Args   map[string]any
	URI    string
}

func (f *FakeClient) Initialize(ctx context.Context) (*InitializeResult, time.Duration, error) {
	f.CallLog = append(f.CallLog, CallRecord{Method: "initialize"})
	if f.InitErr != nil {
		return nil, 0, f.InitErr
	}
	if f.InitResult != nil {
		return f.InitResult, f.InitDuration, nil
	}
	return &InitializeResult{
		ProtocolVersion: ProtocolVersion,
		ServerInfo:      ServerInfo{Name: "fake", Version: "1.0.0"},
	}, f.InitDuration, nil
}

func (f *FakeClient) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	f.CallLog = append(f.CallLog, CallRecord{Method: "tools/list"})
	if f.ToolsErr != nil {
		return nil, f.ToolsErr
	}
	return f.Tools, nil
}

func (f *FakeClient) CallTool(ctx context.Context, name string, args map[string]any) (*RPCResponse, error) {
	f.CallLog = append(f.CallLog, CallRecord{Method: "tools/call", Name: name, Args: args})
	if err, ok := f.ToolErrors[name]; ok {
		return nil, err
	}
	if resp, ok := f.ToolResponses[name]; ok {
		return resp, nil
	}
	return nil, fmt.Errorf("fake: no canned response for tool %q", name)
}

func (f *FakeClient) ListResources(ctx context.Context) ([]Resource, error) {
	f.CallLog = append(f.CallLog, CallRecord{Method: "resources/list"})
	if f.ResourcesErr != nil {
		return nil, f.ResourcesErr
	}
	return f.Resources, nil
}

func (f *FakeClient) ReadResource(ctx context.Context, uri string) ([]ResourceContent, error) {
	f.CallLog = append(f.CallLog, CallRecord{Method: "resources/read", URI: uri})
	if f.ResourceErr != nil {
		return nil, f.ResourceErr
	}
	if c, ok := f.ResourceContent[uri]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("fake: no canned content for uri %q", uri)
}

// NewFakeToolCallResponse is a helper that wraps a ToolCallResult in an
// RPCResponse for use in FakeClient.ToolResponses.
func NewFakeToolCallResponse(id int, result ToolCallResult) *RPCResponse {
	idBytes, _ := json.Marshal(id)
	resBytes, _ := json.Marshal(result)
	return &RPCResponse{JSONRPC: "2.0", ID: idBytes, Result: resBytes}
}
