package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://api-discovery.mintlify.com"
	searchEndpoint = "/v1/search"
	defaultTimeout = 3 * time.Second
)

var (
	ErrUnauthorized = errors.New("unauthorized: invalid or missing API key")
	ErrNotFound     = errors.New("not found: check your domain configuration")
)

type Client struct {
	httpClient *http.Client
	apiKey     string
	domain     string
	baseURL    string
}

func NewClient(apiKey, domain string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		apiKey:     apiKey,
		domain:     domain,
		baseURL:    defaultBaseURL,
	}
}

func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	reqBody := SearchRequest{
		Query:    query,
		PageSize: limit,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	url := c.baseURL + searchEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-Mintlify-Domain", c.domain)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // Non-actionable on read close.

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrUnauthorized
	case http.StatusNotFound:
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var raw searchResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return raw.Results, nil
}

func (c *Client) Ping(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	_, err := c.Search(ctx, "test", 1)
	if err != nil {
		return 0, err
	}
	return time.Since(start), nil
}
