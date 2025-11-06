package tools

import (
	"context"
	"fmt"

	"dev.helix.code/internal/tools/web"
)

// WebFetchTool implements web content fetching
type WebFetchTool struct {
	registry *ToolRegistry
}

func (t *WebFetchTool) Name() string { return "web_fetch" }

func (t *WebFetchTool) Description() string {
	return "Fetch content from a URL"
}

func (t *WebFetchTool) Category() ToolCategory {
	return CategoryWeb
}

func (t *WebFetchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to fetch",
			},
			"parse_markdown": map[string]interface{}{
				"type":        "boolean",
				"description": "Parse HTML to markdown (default: true)",
			},
			"follow_redirects": map[string]interface{}{
				"type":        "boolean",
				"description": "Follow redirects (default: true)",
			},
		},
		Required:    []string{"url"},
		Description: "Fetch content from a URL",
	}
}

func (t *WebFetchTool) Validate(params map[string]interface{}) error {
	if _, ok := params["url"]; !ok {
		return fmt.Errorf("url is required")
	}
	return nil
}

func (t *WebFetchTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	url := params["url"].(string)
	parseMarkdown := true

	if val, ok := params["parse_markdown"].(bool); ok {
		parseMarkdown = val
	}

	if parseMarkdown {
		markdown, metadata, err := t.registry.web.FetchAndParse(ctx, url)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"markdown": markdown,
			"metadata": metadata,
		}, nil
	}

	opts := web.FetchOptions{}

	return t.registry.web.Fetch(ctx, url, opts)
}

// WebSearchTool implements web search
type WebSearchTool struct {
	registry *ToolRegistry
}

func (t *WebSearchTool) Name() string { return "web_search" }

func (t *WebSearchTool) Description() string {
	return "Search the web for information"
}

func (t *WebSearchTool) Category() ToolCategory {
	return CategoryWeb
}

func (t *WebSearchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 10)",
			},
			"provider": map[string]interface{}{
				"type":        "string",
				"description": "Search provider (google, bing, duckduckgo)",
			},
		},
		Required:    []string{"query"},
		Description: "Search the web for information",
	}
}

func (t *WebSearchTool) Validate(params map[string]interface{}) error {
	if _, ok := params["query"]; !ok {
		return fmt.Errorf("query is required")
	}
	return nil
}

func (t *WebSearchTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query := params["query"].(string)
	opts := web.SearchOptions{
		MaxResults: 10,
	}

	if maxResults, ok := params["max_results"].(int); ok {
		opts.MaxResults = maxResults
	}

	if provider, ok := params["provider"].(string); ok {
		switch provider {
		case "google":
			opts.Provider = web.ProviderGoogle
		case "bing":
			opts.Provider = web.ProviderBing
		case "duckduckgo":
			opts.Provider = web.ProviderDuckDuckGo
		}
	}

	return t.registry.web.Search(ctx, query, opts)
}
