package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// SearchProvider identifies search provider
type SearchProvider int

const (
	ProviderGoogle SearchProvider = iota
	ProviderBing
	ProviderDuckDuckGo
)

// String returns the string representation of the provider
func (sp SearchProvider) String() string {
	switch sp {
	case ProviderGoogle:
		return "google"
	case ProviderBing:
		return "bing"
	case ProviderDuckDuckGo:
		return "duckduckgo"
	default:
		return "unknown"
	}
}

// SearchEngine manages search operations
type SearchEngine struct {
	httpClient   *http.Client
	cacheManager *CacheManager
	rateLimiter  *RateLimiter
	config       *Config
	providers    map[SearchProvider]Provider
}

// SearchOptions configures search behavior
type SearchOptions struct {
	Provider     SearchProvider
	MaxResults   int
	Language     string
	Country      string
	SafeSearch   bool
	TimeRange    TimeRange
	FileType     string
	Site         string
}

// TimeRange filters by time
type TimeRange string

const (
	TimeAny   TimeRange = ""
	TimeDay   TimeRange = "d"
	TimeWeek  TimeRange = "w"
	TimeMonth TimeRange = "m"
	TimeYear  TimeRange = "y"
)

// SearchResult contains search results
type SearchResult struct {
	Query        string
	Provider     SearchProvider
	Results      []SearchItem
	TotalResults int64
	SearchTime   float64
	Timestamp    time.Time
}

// SearchItem represents a single search result
type SearchItem struct {
	Title      string
	URL        string
	Snippet    string
	DisplayURL string
	Position   int
	Favicon    string
	Metadata   map[string]interface{}
}

// Provider interface for search providers
type Provider interface {
	Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error)
	Name() string
}

// NewSearchEngine creates a new search engine
func NewSearchEngine(httpClient *http.Client, cacheManager *CacheManager, rateLimiter *RateLimiter, config *Config) *SearchEngine {
	se := &SearchEngine{
		httpClient:   httpClient,
		cacheManager: cacheManager,
		rateLimiter:  rateLimiter,
		config:       config,
		providers:    make(map[SearchProvider]Provider),
	}

	// Initialize providers
	if config.GoogleAPIKey != "" && config.GoogleCSEID != "" {
		se.providers[ProviderGoogle] = &GoogleProvider{
			apiKey:     config.GoogleAPIKey,
			cseID:      config.GoogleCSEID,
			httpClient: httpClient,
		}
	}

	if config.BingAPIKey != "" {
		se.providers[ProviderBing] = &BingProvider{
			apiKey:     config.BingAPIKey,
			httpClient: httpClient,
		}
	}

	// DuckDuckGo doesn't require API key
	se.providers[ProviderDuckDuckGo] = &DuckDuckGoProvider{
		httpClient: httpClient,
	}

	return se
}

// Search executes a search query
func (se *SearchEngine) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
	// Set defaults
	if opts.MaxResults == 0 {
		opts.MaxResults = se.config.MaxSearchResults
	}
	if opts.Provider == 0 && opts.Provider != ProviderGoogle {
		opts.Provider = se.config.DefaultProvider
	}

	// Check cache
	cacheKey := fmt.Sprintf("search:%s:%d:%s", opts.Provider.String(), opts.MaxResults, query)
	if se.cacheManager != nil {
		if cached, ok := se.cacheManager.Get(cacheKey); ok {
			var result SearchResult
			if err := json.Unmarshal(cached, &result); err == nil {
				return &result, nil
			}
		}
	}

	// Get provider
	provider := se.providers[opts.Provider]
	if provider == nil {
		return nil, fmt.Errorf("provider %s not available", opts.Provider.String())
	}

	// Check rate limit
	if se.rateLimiter != nil {
		if err := se.rateLimiter.Wait(ctx, opts.Provider.String()); err != nil {
			return nil, fmt.Errorf("rate limit: %w", err)
		}
	}

	// Execute search
	result, err := provider.Search(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Cache result
	if se.cacheManager != nil {
		if data, err := json.Marshal(result); err == nil {
			se.cacheManager.Set(cacheKey, data)
		}
	}

	return result, nil
}

// GoogleProvider implements Google Custom Search API
type GoogleProvider struct {
	apiKey     string
	cseID      string
	httpClient *http.Client
}

// Name returns the provider name
func (gp *GoogleProvider) Name() string {
	return "google"
}

// Search implements Provider
func (gp *GoogleProvider) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
	// Build request URL
	searchURL := gp.buildSearchURL(query, opts)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := gp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp GoogleSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return gp.convertResponse(&apiResp, query), nil
}

// buildSearchURL builds the Google search URL
func (gp *GoogleProvider) buildSearchURL(query string, opts SearchOptions) string {
	params := url.Values{}
	params.Set("key", gp.apiKey)
	params.Set("cx", gp.cseID)
	params.Set("q", query)

	if opts.MaxResults > 0 {
		params.Set("num", strconv.Itoa(opts.MaxResults))
	}
	if opts.Language != "" {
		params.Set("lr", "lang_"+opts.Language)
	}
	if opts.Country != "" {
		params.Set("cr", "country"+opts.Country)
	}
	if opts.SafeSearch {
		params.Set("safe", "active")
	}
	if opts.FileType != "" {
		params.Set("fileType", opts.FileType)
	}
	if opts.Site != "" {
		params.Set("siteSearch", opts.Site)
	}

	return fmt.Sprintf("https://www.googleapis.com/customsearch/v1?%s", params.Encode())
}

// GoogleSearchResponse represents Google API response
type GoogleSearchResponse struct {
	Items []struct {
		Title       string `json:"title"`
		Link        string `json:"link"`
		Snippet     string `json:"snippet"`
		DisplayLink string `json:"displayLink"`
	} `json:"items"`
	SearchInformation struct {
		TotalResults string  `json:"totalResults"`
		SearchTime   float64 `json:"searchTime"`
	} `json:"searchInformation"`
}

// convertResponse converts Google response to SearchResult
func (gp *GoogleProvider) convertResponse(apiResp *GoogleSearchResponse, query string) *SearchResult {
	result := &SearchResult{
		Query:      query,
		Provider:   ProviderGoogle,
		Results:    make([]SearchItem, 0, len(apiResp.Items)),
		SearchTime: apiResp.SearchInformation.SearchTime,
		Timestamp:  time.Now(),
	}

	if total, err := strconv.ParseInt(apiResp.SearchInformation.TotalResults, 10, 64); err == nil {
		result.TotalResults = total
	}

	for i, item := range apiResp.Items {
		result.Results = append(result.Results, SearchItem{
			Title:      item.Title,
			URL:        item.Link,
			Snippet:    item.Snippet,
			DisplayURL: item.DisplayLink,
			Position:   i + 1,
		})
	}

	return result
}

// BingProvider implements Bing Web Search API
type BingProvider struct {
	apiKey     string
	httpClient *http.Client
}

// Name returns the provider name
func (bp *BingProvider) Name() string {
	return "bing"
}

// Search implements Provider
func (bp *BingProvider) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
	searchURL := bp.buildSearchURL(query, opts)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	// Bing requires subscription key header
	req.Header.Set("Ocp-Apim-Subscription-Key", bp.apiKey)

	resp, err := bp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var apiResp BingSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return bp.convertResponse(&apiResp, query), nil
}

// buildSearchURL builds the Bing search URL
func (bp *BingProvider) buildSearchURL(query string, opts SearchOptions) string {
	params := url.Values{}
	params.Set("q", query)

	if opts.MaxResults > 0 {
		params.Set("count", strconv.Itoa(opts.MaxResults))
	}
	if opts.SafeSearch {
		params.Set("safeSearch", "Strict")
	}

	return fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?%s", params.Encode())
}

// BingSearchResponse represents Bing API response
type BingSearchResponse struct {
	WebPages struct {
		Value []struct {
			Name       string `json:"name"`
			URL        string `json:"url"`
			Snippet    string `json:"snippet"`
			DisplayURL string `json:"displayUrl"`
		} `json:"value"`
		TotalEstimatedMatches int64 `json:"totalEstimatedMatches"`
	} `json:"webPages"`
}

// convertResponse converts Bing response to SearchResult
func (bp *BingProvider) convertResponse(apiResp *BingSearchResponse, query string) *SearchResult {
	result := &SearchResult{
		Query:        query,
		Provider:     ProviderBing,
		Results:      make([]SearchItem, 0, len(apiResp.WebPages.Value)),
		TotalResults: apiResp.WebPages.TotalEstimatedMatches,
		Timestamp:    time.Now(),
	}

	for i, item := range apiResp.WebPages.Value {
		result.Results = append(result.Results, SearchItem{
			Title:      item.Name,
			URL:        item.URL,
			Snippet:    item.Snippet,
			DisplayURL: item.DisplayURL,
			Position:   i + 1,
		})
	}

	return result
}

// DuckDuckGoProvider implements DuckDuckGo HTML search
type DuckDuckGoProvider struct {
	httpClient *http.Client
}

// Name returns the provider name
func (ddg *DuckDuckGoProvider) Name() string {
	return "duckduckgo"
}

// Search implements Provider
func (ddg *DuckDuckGoProvider) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
	// DuckDuckGo HTML search
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	// Set user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; HelixCode/1.0)")

	resp, err := ddg.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed: status %d", resp.StatusCode)
	}

	// Parse HTML
	return ddg.parseHTML(resp.Body, query, opts.MaxResults)
}

// parseHTML parses DuckDuckGo HTML results
func (ddg *DuckDuckGoProvider) parseHTML(body io.Reader, query string, maxResults int) (*SearchResult, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	result := &SearchResult{
		Query:     query,
		Provider:  ProviderDuckDuckGo,
		Results:   []SearchItem{},
		Timestamp: time.Now(),
	}

	// Extract results
	var extract func(*html.Node)
	position := 1
	extract = func(n *html.Node) {
		if maxResults > 0 && len(result.Results) >= maxResults {
			return
		}

		if n.Type == html.ElementNode && n.Data == "div" {
			// Check if this is a result div
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "result") {
					item := ddg.extractResultItem(n, position)
					if item != nil {
						result.Results = append(result.Results, *item)
						position++
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	result.TotalResults = int64(len(result.Results))
	return result, nil
}

// extractResultItem extracts a single result item from HTML node
func (ddg *DuckDuckGoProvider) extractResultItem(n *html.Node, position int) *SearchItem {
	item := &SearchItem{
		Position: position,
	}

	var extract func(*html.Node)
	extract = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if node.Data == "a" {
				for _, attr := range node.Attr {
					if attr.Key == "class" && strings.Contains(attr.Val, "result__a") {
						// Extract title
						if node.FirstChild != nil {
							item.Title = getNodeText(node)
						}
					}
					if attr.Key == "href" {
						item.URL = attr.Val
					}
				}
			} else if node.Data == "a" {
				for _, attr := range node.Attr {
					if attr.Key == "class" && strings.Contains(attr.Val, "result__snippet") {
						item.Snippet = getNodeText(node)
					}
				}
			}
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(n)

	if item.Title != "" && item.URL != "" {
		return item
	}
	return nil
}

// getNodeText extracts text content from HTML node
func getNodeText(n *html.Node) string {
	var text strings.Builder
	var extract func(*html.Node)
	extract = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(n)
	return strings.TrimSpace(text.String())
}
