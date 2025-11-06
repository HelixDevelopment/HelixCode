package web

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// WebTools provides web search and fetching capabilities
type WebTools struct {
	config        *Config
	searchEngine  *SearchEngine
	fetcher       *Fetcher
	parser        *Parser
	cacheManager  *CacheManager
	rateLimiter   *RateLimiter
	httpClient    *http.Client
	mu            sync.RWMutex
}

// Config contains web tools configuration
type Config struct {
	// Search configuration
	DefaultProvider   SearchProvider
	GoogleAPIKey      string
	GoogleCSEID       string
	BingAPIKey        string
	MaxSearchResults  int
	SearchTimeout     time.Duration

	// Fetch configuration
	FetchTimeout      time.Duration
	MaxContentSize    int64
	FollowRedirects   bool
	MaxRedirects      int
	UserAgents        []string

	// Parsing configuration
	RemoveScripts     bool
	RemoveStyles      bool
	RemoveNavigation  bool
	ExtractMetadata   bool

	// Caching configuration
	CacheEnabled      bool
	CacheTTL          time.Duration
	CacheDir          string
	MaxCacheSize      int64

	// Rate limiting
	RateLimitEnabled  bool

	// Security
	BlockedDomains    []string
	AllowPrivateIPs   bool
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider:   ProviderDuckDuckGo,
		MaxSearchResults:  10,
		SearchTimeout:     30 * time.Second,
		FetchTimeout:      30 * time.Second,
		MaxContentSize:    10 * 1024 * 1024, // 10 MB
		FollowRedirects:   true,
		MaxRedirects:      10,
		UserAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		},
		RemoveScripts:    true,
		RemoveStyles:     true,
		RemoveNavigation: true,
		ExtractMetadata:  true,
		CacheEnabled:     true,
		CacheTTL:         15 * time.Minute,
		MaxCacheSize:     100 * 1024 * 1024, // 100 MB
		RateLimitEnabled: true,
		BlockedDomains: []string{
			"*.onion",
			"*.i2p",
		},
		AllowPrivateIPs:  false,
	}
}

// NewWebTools creates a new web tools instance
func NewWebTools(config *Config) (*WebTools, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.FetchTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		},
	}

	// Handle redirects
	if !config.FollowRedirects {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if config.MaxRedirects > 0 {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", config.MaxRedirects)
			}
			return nil
		}
	}

	wt := &WebTools{
		config:     config,
		httpClient: httpClient,
	}

	// Initialize cache manager
	if config.CacheEnabled {
		wt.cacheManager = NewCacheManager(config.CacheDir, config.CacheTTL, config.MaxCacheSize)
	}

	// Initialize rate limiter
	if config.RateLimitEnabled {
		wt.rateLimiter = NewRateLimiter()
	}

	// Initialize parser
	wt.parser = NewParser(config)

	// Initialize fetcher
	wt.fetcher = NewFetcher(httpClient, wt.cacheManager, wt.rateLimiter, config)

	// Initialize search engine
	wt.searchEngine = NewSearchEngine(httpClient, wt.cacheManager, wt.rateLimiter, config)

	return wt, nil
}

// Search performs a web search
func (wt *WebTools) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	return wt.searchEngine.Search(ctx, query, opts)
}

// Fetch fetches content from a URL
func (wt *WebTools) Fetch(ctx context.Context, url string, opts FetchOptions) (*FetchResult, error) {
	if url == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	return wt.fetcher.Fetch(ctx, url, opts)
}

// FetchAndParse fetches content from a URL and parses it to markdown
func (wt *WebTools) FetchAndParse(ctx context.Context, url string) (string, *Metadata, error) {
	// Fetch content
	result, err := wt.Fetch(ctx, url, FetchOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("fetch failed: %w", err)
	}

	// Check if redirect
	if result.Redirect != "" {
		return "", nil, fmt.Errorf("redirected to: %s", result.Redirect)
	}

	// Parse HTML to markdown
	parsed, err := wt.parser.Parse(result.Content, url)
	if err != nil {
		return "", nil, fmt.Errorf("parse failed: %w", err)
	}

	return parsed.Markdown, &parsed.Metadata, nil
}

// SearchAndFetch performs a search and fetches the first result
func (wt *WebTools) SearchAndFetch(ctx context.Context, query string) (string, *Metadata, error) {
	// Search
	searchResult, err := wt.Search(ctx, query, SearchOptions{
		MaxResults: 1,
	})
	if err != nil {
		return "", nil, fmt.Errorf("search failed: %w", err)
	}

	if len(searchResult.Results) == 0 {
		return "", nil, fmt.Errorf("no results found")
	}

	// Fetch first result
	url := searchResult.Results[0].URL
	return wt.FetchAndParse(ctx, url)
}

// ClearCache clears the web tools cache
func (wt *WebTools) ClearCache() error {
	if wt.cacheManager != nil {
		return wt.cacheManager.Clear()
	}
	return nil
}

// GetCacheStats returns cache statistics
func (wt *WebTools) GetCacheStats() *CacheStats {
	if wt.cacheManager != nil {
		return wt.cacheManager.GetStats()
	}
	return nil
}

// Close closes the web tools and releases resources
func (wt *WebTools) Close() error {
	wt.httpClient.CloseIdleConnections()
	if wt.cacheManager != nil {
		return wt.cacheManager.Close()
	}
	return nil
}
