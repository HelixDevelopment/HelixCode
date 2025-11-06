// Package web provides comprehensive web tools for HelixCode, including web search,
// URL fetching, HTML parsing, and content extraction capabilities.
//
// # Overview
//
// The web package enables AI agents to access web content through three main capabilities:
//   - Web Search: Multi-provider search engine support (Google, Bing, DuckDuckGo)
//   - URL Fetching: HTTP/HTTPS content retrieval with caching and rate limiting
//   - HTML Parsing: Convert HTML to clean markdown for LLM consumption
//
// # Architecture
//
// The package is organized into several key components:
//
//   - WebTools: Main coordinator that orchestrates all web operations
//   - SearchEngine: Manages web searches across multiple providers
//   - Fetcher: Handles HTTP requests with caching and rate limiting
//   - Parser: Converts HTML to markdown and extracts metadata
//   - CacheManager: Manages both memory and disk caching with TTL
//   - RateLimiter: Prevents API rate limit violations
//
// # Basic Usage
//
// Create a new WebTools instance with default configuration:
//
//	wt, err := web.NewWebTools(nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer wt.Close()
//
// # Web Search
//
// Perform web searches using multiple providers:
//
//	// DuckDuckGo search (no API key required)
//	result, err := wt.Search(context.Background(), "golang best practices", web.SearchOptions{
//	    Provider:   web.ProviderDuckDuckGo,
//	    MaxResults: 10,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, item := range result.Results {
//	    fmt.Printf("%s - %s\n", item.Title, item.URL)
//	}
//
//	// Google Custom Search (requires API key)
//	config := web.DefaultConfig()
//	config.GoogleAPIKey = "your-api-key"
//	config.GoogleCSEID = "your-cse-id"
//	wt, _ := web.NewWebTools(config)
//
//	result, err := wt.Search(context.Background(), "machine learning", web.SearchOptions{
//	    Provider: web.ProviderGoogle,
//	})
//
//	// Bing Web Search (requires API key)
//	config := web.DefaultConfig()
//	config.BingAPIKey = "your-api-key"
//	wt, _ := web.NewWebTools(config)
//
//	result, err := wt.Search(context.Background(), "cloud computing", web.SearchOptions{
//	    Provider: web.ProviderBing,
//	})
//
// # URL Fetching
//
// Fetch content from URLs with automatic caching:
//
//	result, err := wt.Fetch(context.Background(), "https://example.com", web.FetchOptions{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Status: %d\n", result.Status)
//	fmt.Printf("Content-Type: %s\n", result.ContentType)
//	fmt.Printf("Size: %d bytes\n", result.Size)
//
// Fetch with custom options:
//
//	result, err := wt.Fetch(context.Background(), "https://example.com", web.FetchOptions{
//	    Timeout:   10 * time.Second,
//	    MaxSize:   5 * 1024 * 1024, // 5 MB
//	    UserAgent: "CustomBot/1.0",
//	    Headers: map[string]string{
//	        "Authorization": "Bearer token",
//	    },
//	})
//
// # HTML to Markdown Conversion
//
// Fetch and parse HTML to clean markdown:
//
//	markdown, metadata, err := wt.FetchAndParse(context.Background(), "https://example.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Title: %s\n", metadata.Title)
//	fmt.Printf("Description: %s\n", metadata.Description)
//	fmt.Println("Content:")
//	fmt.Println(markdown)
//
// # Configuration
//
// Configure web tools with custom settings:
//
//	config := &web.Config{
//	    // Search configuration
//	    DefaultProvider:  web.ProviderDuckDuckGo,
//	    MaxSearchResults: 20,
//	    SearchTimeout:    30 * time.Second,
//
//	    // Fetch configuration
//	    FetchTimeout:     30 * time.Second,
//	    MaxContentSize:   10 * 1024 * 1024, // 10 MB
//	    FollowRedirects:  true,
//	    MaxRedirects:     10,
//
//	    // Parsing configuration
//	    RemoveScripts:    true,
//	    RemoveStyles:     true,
//	    RemoveNavigation: true,
//	    ExtractMetadata:  true,
//
//	    // Caching configuration
//	    CacheEnabled:     true,
//	    CacheTTL:         15 * time.Minute,
//	    CacheDir:         "/path/to/cache",
//	    MaxCacheSize:     100 * 1024 * 1024, // 100 MB
//
//	    // Rate limiting
//	    RateLimitEnabled: true,
//
//	    // Security
//	    BlockedDomains: []string{
//	        "*.onion",
//	        "*.i2p",
//	    },
//	    AllowPrivateIPs: false,
//	}
//
//	wt, err := web.NewWebTools(config)
//
// # Caching
//
// The package implements a two-tier caching system:
//
//   - Memory Cache: Fast LRU cache for frequently accessed content
//   - Disk Cache: Persistent storage for larger datasets
//
// Cache operations:
//
//	// Get cache statistics
//	stats := wt.GetCacheStats()
//	fmt.Printf("Hits: %d, Misses: %d\n", stats.Hits.Load(), stats.Misses.Load())
//
//	// Clear cache
//	err := wt.ClearCache()
//
// # Rate Limiting
//
// Rate limiting prevents API abuse and respects provider limits:
//
//   - Google: 10 requests/second, burst 20
//   - Bing: 5 requests/second, burst 10
//   - DuckDuckGo: 2 requests/second, burst 5
//   - Default: 1 request/second, burst 3
//
// Rate limits are automatically applied per provider/domain.
//
// # Multiple Concurrent Fetches
//
// Fetch multiple URLs concurrently with automatic rate limiting:
//
//	urls := []string{
//	    "https://example.com/page1",
//	    "https://example.com/page2",
//	    "https://example.com/page3",
//	}
//
//	results, err := wt.fetcher.FetchMultiple(context.Background(), urls, web.FetchOptions{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for i, result := range results {
//	    fmt.Printf("URL %d: %d bytes\n", i, result.Size)
//	}
//
// # Security
//
// The package includes several security features:
//
//   - URL Validation: Blocks invalid schemes and malformed URLs
//   - Domain Blocking: Prevents access to blocked domains
//   - Private IP Protection: Blocks access to private/loopback IPs
//   - Content Size Limits: Prevents memory exhaustion
//   - Timeout Protection: Prevents hung requests
//
// # Error Handling
//
// The package returns descriptive errors for common scenarios:
//
//	result, err := wt.Fetch(ctx, url, opts)
//	if err != nil {
//	    switch {
//	    case strings.Contains(err.Error(), "404"):
//	        fmt.Println("Page not found")
//	    case strings.Contains(err.Error(), "timeout"):
//	        fmt.Println("Request timed out")
//	    case strings.Contains(err.Error(), "rate limit"):
//	        fmt.Println("Rate limit exceeded")
//	    default:
//	        fmt.Printf("Error: %v\n", err)
//	    }
//	}
//
// # Context Support
//
// All operations support context for cancellation and timeouts:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	result, err := wt.Search(ctx, "query", web.SearchOptions{})
//	if err == context.DeadlineExceeded {
//	    fmt.Println("Search timed out")
//	}
//
// # Performance Considerations
//
// Connection Pooling: The HTTP client uses connection pooling for efficiency:
//   - MaxIdleConns: 100
//   - MaxIdleConnsPerHost: 10
//   - IdleConnTimeout: 90 seconds
//
// Concurrent Operations: Multiple fetches are performed concurrently with
// a semaphore limiting concurrency to 5 simultaneous requests.
//
// Caching Strategy: Content is cached in both memory and disk with a
// 15-minute default TTL. Memory cache uses LRU eviction.
//
// # Testing
//
// The package includes comprehensive tests using httptest.NewServer
// for mocking HTTP responses:
//
//	func TestFetch(t *testing.T) {
//	    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        w.Write([]byte("<html><body>Test</body></html>"))
//	    }))
//	    defer server.Close()
//
//	    wt, _ := web.NewWebTools(nil)
//	    result, err := wt.Fetch(context.Background(), server.URL, web.FetchOptions{})
//	    // assertions...
//	}
//
// # Integration Examples
//
// Search and fetch first result:
//
//	markdown, metadata, err := wt.SearchAndFetch(context.Background(), "golang documentation")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(markdown)
//
// Extract plain text without markdown:
//
//	result, err := wt.Fetch(context.Background(), url, web.FetchOptions{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	text, err := wt.parser.ExtractText(result.Content)
//	fmt.Println(text)
//
// # References
//
// This implementation is inspired by and compatible with:
//   - Qwen Code's web tools architecture
//   - Cline's web fetch capabilities
//   - Standard Go HTTP client patterns
//
// The package follows HelixCode's architectural patterns:
//   - Context-based cancellation
//   - Error wrapping with fmt.Errorf
//   - Interface-based design
//   - Comprehensive testing with testify
package web
