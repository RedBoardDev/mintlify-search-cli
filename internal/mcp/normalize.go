package mcp

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

func FindSearchTool(discovery *Discovery) (*ToolDefinition, error) {
	if discovery == nil {
		return nil, fmt.Errorf("missing discovery response")
	}

	if len(discovery.Capabilities.Tools) == 1 {
		for name, tool := range discovery.Capabilities.Tools {
			if tool.Name == "" {
				tool.Name = name
			}
			return &tool, nil
		}
	}

	for name, tool := range discovery.Capabilities.Tools {
		toolName := tool.Name
		if toolName == "" {
			toolName = name
		}
		if strings.HasPrefix(toolName, "search_") && requiresQuery(tool.InputSchema) {
			tool.Name = toolName
			return &tool, nil
		}
	}

	return nil, fmt.Errorf("no search tool with required query input found")
}

func FindSearchToolFromList(tools []ToolDefinition) (*ToolDefinition, error) {
	if len(tools) == 1 {
		return &tools[0], nil
	}
	for _, tool := range tools {
		if strings.HasPrefix(tool.Name, "search_") && requiresQuery(tool.InputSchema) {
			return &tool, nil
		}
	}
	return nil, fmt.Errorf("no search tool with required query input found")
}

func requiresQuery(schema InputSchema) bool {
	prop, ok := schema.Properties["query"]
	if !ok || prop.Type != "string" {
		return false
	}
	return slices.Contains(schema.Required, "query")
}

func ParseToolCallResult(resp *RPCResponse) (*ToolCallResult, error) {
	var result ToolCallResult
	if err := unmarshalRPCResult(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func NormalizeSearchResults(call *ToolCallResult) []SearchResult {
	if call == nil {
		return nil
	}

	results := make([]SearchResult, 0, len(call.Content))
	for _, block := range call.Content {
		if block.Type != "text" || strings.TrimSpace(block.Text) == "" {
			continue
		}

		result := normalizeTextBlock(block.Text)
		if result.URL == "" {
			continue
		}
		results = append(results, result)
	}
	return results
}

func MarshalMinified(v any) ([]byte, error) {
	return json.Marshal(v)
}

func normalizeTextBlock(raw string) SearchResult {
	lines := strings.Split(raw, "\n")
	result := SearchResult{}
	var contentParts []string
	inContent := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inContent {
				contentParts = append(contentParts, "")
			}
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "Title:"):
			result.Title = cleanText(strings.TrimSpace(strings.TrimPrefix(trimmed, "Title:")))
			inContent = false
		case strings.HasPrefix(trimmed, "Link:"):
			result.URL = strings.TrimSpace(strings.TrimPrefix(trimmed, "Link:"))
			inContent = false
		case strings.HasPrefix(trimmed, "Content:"):
			content := strings.TrimSpace(strings.TrimPrefix(trimmed, "Content:"))
			if content != "" {
				contentParts = append(contentParts, content)
			}
			inContent = true
		case inContent:
			contentParts = append(contentParts, trimmed)
		}
	}

	result.Content = cleanText(strings.Join(contentParts, "\n"))
	if result.Content == "" {
		result.Content = cleanText(raw)
	}
	if result.Title == "" {
		result.Title = result.URL
	}

	return result
}

func cleanText(value string) string {
	value = htmlTagPattern.ReplaceAllString(value, "")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.Join(strings.Fields(value), " ")
}
