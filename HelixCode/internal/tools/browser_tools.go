package tools

import (
	"context"
	"fmt"

	"dev.helix.code/internal/tools/browser"
)

// BrowserLaunchTool launches a browser instance
type BrowserLaunchTool struct {
	registry *ToolRegistry
}

func (t *BrowserLaunchTool) Name() string { return "browser_launch" }

func (t *BrowserLaunchTool) Description() string {
	return "Launch a new browser instance"
}

func (t *BrowserLaunchTool) Category() ToolCategory {
	return CategoryBrowser
}

func (t *BrowserLaunchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"headless": map[string]interface{}{
				"type":        "boolean",
				"description": "Run browser in headless mode (default: true)",
			},
			"width": map[string]interface{}{
				"type":        "integer",
				"description": "Browser window width (default: 1280)",
			},
			"height": map[string]interface{}{
				"type":        "integer",
				"description": "Browser window height (default: 720)",
			},
		},
		Required:    []string{},
		Description: "Launch a new browser instance",
	}
}

func (t *BrowserLaunchTool) Validate(params map[string]interface{}) error {
	return nil
}

func (t *BrowserLaunchTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	opts := &browser.LaunchOptions{
		Headless: true,
		Width:    1280,
		Height:   720,
	}

	if headless, ok := params["headless"].(bool); ok {
		opts.Headless = headless
	}

	if width, ok := params["width"].(int); ok {
		opts.Width = width
	}

	if height, ok := params["height"].(int); ok {
		opts.Height = height
	}

	return t.registry.browser.LaunchBrowser(ctx, opts)
}

// BrowserNavigateTool navigates to a URL
type BrowserNavigateTool struct {
	registry *ToolRegistry
}

func (t *BrowserNavigateTool) Name() string { return "browser_navigate" }

func (t *BrowserNavigateTool) Description() string {
	return "Navigate browser to a URL"
}

func (t *BrowserNavigateTool) Category() ToolCategory {
	return CategoryBrowser
}

func (t *BrowserNavigateTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"browser_id": map[string]interface{}{
				"type":        "string",
				"description": "Browser instance ID",
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to navigate to",
			},
		},
		Required:    []string{"browser_id", "url"},
		Description: "Navigate browser to a URL",
	}
}

func (t *BrowserNavigateTool) Validate(params map[string]interface{}) error {
	if _, ok := params["browser_id"]; !ok {
		return fmt.Errorf("browser_id is required")
	}
	if _, ok := params["url"]; !ok {
		return fmt.Errorf("url is required")
	}
	return nil
}

func (t *BrowserNavigateTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	browserID := params["browser_id"].(string)
	url := params["url"].(string)

	return nil, t.registry.browser.Navigate(ctx, browserID, url)
}

// BrowserScreenshotTool takes a screenshot
type BrowserScreenshotTool struct {
	registry *ToolRegistry
}

func (t *BrowserScreenshotTool) Name() string { return "browser_screenshot" }

func (t *BrowserScreenshotTool) Description() string {
	return "Take a screenshot of the browser window"
}

func (t *BrowserScreenshotTool) Category() ToolCategory {
	return CategoryBrowser
}

func (t *BrowserScreenshotTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"browser_id": map[string]interface{}{
				"type":        "string",
				"description": "Browser instance ID",
			},
			"format": map[string]interface{}{
				"type":        "string",
				"description": "Image format (png, jpeg)",
			},
			"quality": map[string]interface{}{
				"type":        "integer",
				"description": "Image quality (0-100)",
			},
			"annotate": map[string]interface{}{
				"type":        "boolean",
				"description": "Annotate interactive elements (default: false)",
			},
		},
		Required:    []string{"browser_id"},
		Description: "Take a screenshot of the browser window",
	}
}

func (t *BrowserScreenshotTool) Validate(params map[string]interface{}) error {
	if _, ok := params["browser_id"]; !ok {
		return fmt.Errorf("browser_id is required")
	}
	return nil
}

func (t *BrowserScreenshotTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	browserID := params["browser_id"].(string)
	opts := &browser.ScreenshotOptions{
		Format:  browser.FormatPNG,
		Quality: 90,
	}

	if format, ok := params["format"].(string); ok {
		if format == "jpeg" {
			opts.Format = browser.FormatJPEG
		}
	}

	if quality, ok := params["quality"].(int); ok {
		opts.Quality = quality
	}

	annotate := false
	if val, ok := params["annotate"].(bool); ok {
		annotate = val
	}

	if annotate {
		screenshot, elements, err := t.registry.browser.TakeAnnotatedScreenshot(ctx, browserID, opts)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"screenshot": screenshot,
			"elements":   elements,
		}, nil
	}

	return t.registry.browser.TakeScreenshot(ctx, browserID, opts)
}

// BrowserCloseTool closes a browser instance
type BrowserCloseTool struct {
	registry *ToolRegistry
}

func (t *BrowserCloseTool) Name() string { return "browser_close" }

func (t *BrowserCloseTool) Description() string {
	return "Close a browser instance"
}

func (t *BrowserCloseTool) Category() ToolCategory {
	return CategoryBrowser
}

func (t *BrowserCloseTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"browser_id": map[string]interface{}{
				"type":        "string",
				"description": "Browser instance ID",
			},
		},
		Required:    []string{"browser_id"},
		Description: "Close a browser instance",
	}
}

func (t *BrowserCloseTool) Validate(params map[string]interface{}) error {
	if _, ok := params["browser_id"]; !ok {
		return fmt.Errorf("browser_id is required")
	}
	return nil
}

func (t *BrowserCloseTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	browserID := params["browser_id"].(string)
	return nil, t.registry.browser.CloseBrowser(browserID)
}
