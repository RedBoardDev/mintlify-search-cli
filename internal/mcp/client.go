package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const defaultTimeout = 5 * time.Second

type Client struct {
	httpClient *http.Client
	baseURL    string
	nextID     atomic.Int64
}

func NewClient(mcpURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		baseURL:    mcpURL,
	}
}

func (c *Client) Discover(ctx context.Context) (*Discovery, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating discovery request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing discovery request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discovery failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out Discovery
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("parsing discovery response: %w", err)
	}
	return &out, nil
}

func (c *Client) Initialize(ctx context.Context) (*InitializeResult, time.Duration, error) {
	start := time.Now()
	resp, err := c.doRPC(ctx, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "msc",
			"version": "0.1.0",
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

func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*RPCResponse, error) {
	return c.doRPC(ctx, "tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	})
}

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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating %s request: %w", method, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing %s request: %w", method, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s failed with status %d: %s", method, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	parsed, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("parsing %s response: %w", method, err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("mcp error %d: %s", parsed.Error.Code, parsed.Error.Message)
	}
	return parsed, nil
}

func decodeResponse(resp *http.Response) (*RPCResponse, error) {
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		var out RPCResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}
		return &out, nil
	}

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	var (
		event     string
		dataLines []string
	)

	flushEvent := func() (*RPCResponse, bool, error) {
		if event != "message" || len(dataLines) == 0 {
			event = ""
			dataLines = nil
			return nil, false, nil
		}

		var out RPCResponse
		if err := json.Unmarshal([]byte(strings.Join(dataLines, "\n")), &out); err != nil {
			return nil, false, err
		}
		return &out, true, nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if out, ok, err := flushEvent(); err != nil {
				return nil, err
			} else if ok {
				return out, nil
			}
			continue
		}

		switch {
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimPrefix(line, "data:")
			dataLines = append(dataLines, strings.TrimPrefix(data, " "))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if out, ok, err := flushEvent(); err != nil {
		return nil, err
	} else if ok {
		return out, nil
	}

	return nil, io.ErrUnexpectedEOF
}

func unmarshalRPCResult(resp *RPCResponse, target any) error {
	if len(resp.Result) == 0 {
		return fmt.Errorf("missing result payload")
	}
	if err := json.Unmarshal(resp.Result, target); err != nil {
		return fmt.Errorf("decoding result payload: %w", err)
	}
	return nil
}
