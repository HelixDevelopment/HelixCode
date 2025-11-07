package mentions

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// URLMentionHandler handles @url mentions
type URLMentionHandler struct {
	client *http.Client
	cache  map[string]*MentionContext
}

// NewURLMentionHandler creates a new URL mention handler
func NewURLMentionHandler() *URLMentionHandler {
	return &URLMentionHandler{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: make(map[string]*MentionContext),
	}
}

// Type returns the mention type
func (h *URLMentionHandler) Type() MentionType {
	return MentionTypeURL
}

// CanHandle checks if this handler can handle the mention
func (h *URLMentionHandler) CanHandle(mention string) bool {
	return strings.HasPrefix(mention, "@url[") || strings.HasPrefix(mention, "@url(")
}

// Resolve resolves the URL mention and returns its content
func (h *URLMentionHandler) Resolve(ctx context.Context, target string, options map[string]string) (*MentionContext, error) {
	if target == "" {
		return nil, fmt.Errorf("URL target cannot be empty")
	}

	// Ensure URL has scheme
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}

	// Check cache first (with 15-minute TTL)
	if cached, exists := h.cache[target]; exists {
		if time.Since(cached.ResolvedAt) < 15*time.Minute {
			return cached, nil
		}
		// Cache expired, delete it
		delete(h.cache, target)
	}

	// Fetch URL content
	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "HelixCode/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Read content
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Process content based on Content-Type
	contentType := resp.Header.Get("Content-Type")
	var content string
	var metadata map[string]interface{}

	if strings.Contains(contentType, "text/html") {
		// Extract text from HTML
		content, metadata = h.extractHTMLContent(string(body))
	} else if strings.Contains(contentType, "application/json") {
		content = string(body)
		metadata = map[string]interface{}{
			"content_type": "json",
		}
	} else {
		// Plain text or other
		content = string(body)
		metadata = map[string]interface{}{
			"content_type": contentType,
		}
	}

	// Calculate token count
	tokenCount := len(content) / 4

	mentionCtx := &MentionContext{
		Type:       MentionTypeURL,
		Target:     target,
		Content:    content,
		TokenCount: tokenCount,
		Metadata: map[string]interface{}{
			"status_code":  resp.StatusCode,
			"content_type": contentType,
			"url":          target,
		},
		ResolvedAt: time.Now(),
	}

	// Merge additional metadata
	if metadata != nil {
		for k, v := range metadata {
			mentionCtx.Metadata[k] = v
		}
	}

	// Cache the result
	h.cache[target] = mentionCtx

	return mentionCtx, nil
}

// extractHTMLContent extracts text content from HTML
func (h *URLMentionHandler) extractHTMLContent(html string) (string, map[string]interface{}) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html, nil
	}

	metadata := make(map[string]interface{})

	// Extract title
	title := doc.Find("title").First().Text()
	if title != "" {
		metadata["title"] = strings.TrimSpace(title)
	}

	// Extract meta description
	desc, exists := doc.Find("meta[name='description']").Attr("content")
	if exists {
		metadata["description"] = desc
	}

	// Extract Open Graph data
	ogTitle, exists := doc.Find("meta[property='og:title']").Attr("content")
	if exists {
		metadata["og_title"] = ogTitle
	}

	ogDesc, exists := doc.Find("meta[property='og:description']").Attr("content")
	if exists {
		metadata["og_description"] = ogDesc
	}

	// Remove script and style elements
	doc.Find("script, style, nav, footer, header, aside").Remove()

	// Extract main content
	var content strings.Builder

	// Try to find main content area
	mainContent := doc.Find("main, article, .content, #content")
	if mainContent.Length() > 0 {
		mainContent.Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				content.WriteString(text)
				content.WriteString("\n\n")
			}
		})
	} else {
		// Fall back to body
		bodyText := doc.Find("body").Text()
		content.WriteString(strings.TrimSpace(bodyText))
	}

	// Clean up whitespace
	result := strings.Join(strings.Fields(content.String()), " ")

	// Limit length
	if len(result) > 10000 {
		result = result[:10000] + "... (content truncated)"
	}

	return result, metadata
}

// ClearCache clears the URL cache
func (h *URLMentionHandler) ClearCache() {
	h.cache = make(map[string]*MentionContext)
}
