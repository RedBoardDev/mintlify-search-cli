// Package mcp implements a JSON-RPC 2.0 client for Mintlify MCP servers
// (Streamable HTTP transport, SSE-formatted responses).
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// ClientVersion is reported in initialize.clientInfo. Overridden at build time
// via -ldflags "-X ..../internal/mcp.ClientVersion=vX.Y.Z".
var ClientVersion = "dev"

const (
	maxRetries       = 2
	retryBackoffBase = 100 * time.Millisecond
	sseBufferMax     = 2 * 1024 * 1024 // 2 MiB per SSE data line
)

// Client is the concrete MCPClient implementation.
type Client struct {
	httpClient *http.Client
	baseURL    string
	nextID     atomic.Int64
	logger     func(format string, args ...any) // nil when verbose disabled
}

// Option configures a Client at construction time.
type Option func(*Client)

// WithHTTPClient injects a custom *http.Client (useful in tests or when the
// caller wants to control transport-level settings like proxy or TLS).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithLogger enables verbose logging of HTTP round-trips. The callback is
// invoked with Printf-style arguments for each request and response.
func WithLogger(fn func(format string, args ...any)) Option {
	return func(c *Client) { c.logger = fn }
}

// NewClient builds a Client pointing at the given MCP endpoint URL. The
// default HTTP client has a 30s overall timeout; per-call deadlines should be
// enforced by the caller via context.
func NewClient(mcpURL string, opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    mcpURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Initialize performs the MCP initialize handshake and returns the negotiated
// protocol version and server info, plus the round-trip duration.
func (c *Client) Initialize(ctx context.Context) (*InitializeResult, time.Duration, error) {
	start := time.Now()
	resp, err := c.doRPC(ctx, "initialize", map[string]any{
		"protocolVersion": ProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    ClientName,
			"version": ClientVersion,
		},
	})
	if err != nil {
		return nil, 0, err
	}

	var result InitializeResult
	if err := unmarshalRPCResult(resp, &result); err != nil {
		return nil, 0, err
	}
	return &result, time.Since(start), nil
}

// ListTools fetches the server's tool catalog via JSON-RPC tools/list.
func (c *Client) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	resp, err := c.doRPC(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var result ToolsListResult
	if err := unmarshalRPCResult(resp, &result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

// CallTool invokes a named tool with arbitrary argument map.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*RPCResponse, error) {
	return c.doRPC(ctx, "tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	})
}

// ListResources fetches the server's resource catalog via JSON-RPC resources/list.
func (c *Client) ListResources(ctx context.Context) ([]Resource, error) {
	resp, err := c.doRPC(ctx, "resources/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var result ResourcesListResult
	if err := unmarshalRPCResult(resp, &result); err != nil {
		return nil, err
	}
	return result.Resources, nil
}

// ReadResource fetches a resource's contents by URI.
func (c *Client) ReadResource(ctx context.Context, uri string) ([]ResourceContent, error) {
	resp, err := c.doRPC(ctx, "resources/read", map[string]any{"uri": uri})
	if err != nil {
		return nil, err
	}
	var result ResourceReadResult
	if err := unmarshalRPCResult(resp, &result); err != nil {
		return nil, err
	}
	return result.Contents, nil
}

// doRPC sends a single JSON-RPC request and returns the response. It retries
// up to maxRetries times on 5xx status codes and transport errors, with a
// linear backoff. Context cancellation short-circuits retries.
func (c *Client) doRPC(ctx context.Context, method string, params any) (*RPCResponse, error) {
	id := int(c.nextID.Add(1))
	payload, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return nil, fmt.Errorf("encoding %s request: %w", method, err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, time.Duration(attempt)*retryBackoffBase); err != nil {
				return nil, err
			}
		}

		resp, retriable, err := c.attemptRPC(ctx, method, payload)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !retriable {
			return nil, err
		}
	}
	return nil, fmt.Errorf("%s: exhausted retries: %w", method, lastErr)
}

// attemptRPC performs a single HTTP round-trip. The boolean indicates whether
// a failure is retriable (5xx or transport error).
func (c *Client) attemptRPC(ctx context.Context, method string, payload []byte) (*RPCResponse, bool, error) {
	if c.logger != nil {
		c.logger("POST %s method=%s bytes=%d", c.baseURL, method, len(payload))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, false, fmt.Errorf("creating %s request: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("executing %s request: %w", method, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if c.logger != nil {
		c.logger("HTTP %d %s content-type=%s", resp.StatusCode, method, resp.Header.Get("Content-Type"))
	}

	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(resp.Body)
		return nil, true, fmt.Errorf("%s failed with status %d: %s", method, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("%s failed with status %d: %s", method, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	parsed, err := decodeResponse(resp)
	if err != nil {
		return nil, false, fmt.Errorf("parsing %s response: %w", method, err)
	}
	if parsed.Error != nil {
		return nil, false, fmt.Errorf("mcp error %d: %s", parsed.Error.Code, parsed.Error.Message)
	}
	return parsed, false, nil
}

// decodeResponse reads either a direct JSON body or an SSE stream and returns
// the first non-empty RPC response. SSE streams are robust to batch responses
// (multiple messages) — for single-call usage we return the first one.
func decodeResponse(resp *http.Response) (*RPCResponse, error) {
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		var out RPCResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}
		return &out, nil
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), sseBufferMax)

	var dataLines []string
	flush := func() (*RPCResponse, bool, error) {
		if len(dataLines) == 0 {
			return nil, false, nil
		}
		joined := strings.Join(dataLines, "\n")
		dataLines = nil
		var out RPCResponse
		if err := json.Unmarshal([]byte(joined), &out); err != nil {
			return nil, false, err
		}
		return &out, true, nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if out, ok, err := flush(); err != nil {
				return nil, err
			} else if ok {
				return out, nil
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
		// Silently ignore other SSE field names (event:, id:, retry:, comments).
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if out, ok, err := flush(); err != nil {
		return nil, err
	} else if ok {
		return out, nil
	}
	return nil, io.ErrUnexpectedEOF
}

func unmarshalRPCResult(resp *RPCResponse, target any) error {
	if len(resp.Result) == 0 {
		return errors.New("missing result payload")
	}
	if err := json.Unmarshal(resp.Result, target); err != nil {
		return fmt.Errorf("decoding result payload: %w", err)
	}
	return nil
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
