# Web Tools - Technical Design

## Overview

Web tools provide web search and URL fetching capabilities with HTML-to-markdown conversion, caching, rate limiting, and proxy support. Enables AI agents to access web content and search results.

## Architecture

### Component Diagram

```
┌────────────────────────────────────────────────────────────┐
│                 WebToolsCoordinator                         │
│  - Orchestrates web operations                             │
│  - Manages search and fetch                                │
└────────────┬───────────────────────────────┬───────────────┘
             │                               │
             ▼                               ▼
┌────────────────────────┐      ┌────────────────────────┐
│    SearchEngine        │      │      Fetcher           │
│  - Multi-provider      │      │  - HTTP requests       │
│  - Query execution     │      │  - Content download    │
└──────────┬─────────────┘      └────────┬───────────────┘
           │                              │
           ▼                              ▼
┌────────────────────────┐      ┌────────────────────────┐
│   SearchProvider       │      │      Parser            │
│  - Google              │      │  - HTML to Markdown    │
│  - Bing                │      │  - Content extraction  │
│  - DuckDuckGo          │      │  - Metadata parsing    │
└────────────────────────┘      └────────────────────────┘
           │                              │
           └──────────────┬───────────────┘
                          ▼
                  ┌──────────────┐
                  │ CacheManager │
                  │  - 15min TTL │
                  │  - Disk cache│
                  └──────┬───────┘
                         │
                         ▼
                  ┌──────────────┐
                  │ RateLimiter  │
                  │  - Per-engine│
                  │  - Backoff   │
                  └──────────────┘
```

### Core Components

#### 1. WebToolsCoordinator

```go
package webtools

import (
    "context"
    "time"
)

// WebToolsCoordinator manages web operations
type WebToolsCoordinator struct {
    searchEngine  *SearchEngine
    fetcher       *Fetcher
    cacheManager  *CacheManager
    rateLimiter   *RateLimiter
    config        *Config
}

// NewWebToolsCoordinator creates a new coordinator
func NewWebToolsCoordinator(opts ...Option) *WebToolsCoordinator {
    wtc := &WebToolsCoordinator{
        searchEngine: NewSearchEngine(),
        fetcher:      NewFetcher(),
        cacheManager: NewCacheManager(),
        rateLimiter:  NewRateLimiter(),
        config:       DefaultConfig(),
    }

    for _, opt := range opts {
        opt(wtc)
    }

    return wtc
}

// Search performs a web search
func (wtc *WebToolsCoordinator) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error)

// Fetch fetches content from a URL
func (wtc *WebToolsCoordinator) Fetch(ctx context.Context, url string, opts FetchOptions) (*FetchResult, error)

// FetchAndParse fetches and parses content to markdown
func (wtc *WebToolsCoordinator) FetchAndParse(ctx context.Context, url string) (string, error)
```

#### 2. SearchEngine

```go
// SearchEngine manages search operations
type SearchEngine struct {
    providers    map[SearchProvider]Provider
    rateLimiter  *RateLimiter
    cacheManager *CacheManager
}

// Search executes a search query
func (se *SearchEngine) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
    // Check cache
    if cached, ok := se.cacheManager.GetSearch(query); ok {
        return cached, nil
    }

    // Get provider
    provider := se.providers[opts.Provider]
    if provider == nil {
        provider = se.providers[ProviderGoogle]
    }

    // Check rate limit
    if err := se.rateLimiter.Wait(ctx, opts.Provider); err != nil {
        return nil, fmt.Errorf("rate limit: %w", err)
    }

    // Execute search
    result, err := provider.Search(ctx, query, opts)
    if err != nil {
        return nil, fmt.Errorf("search: %w", err)
    }

    // Cache result
    se.cacheManager.SetSearch(query, result)

    return result, nil
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

// SearchProvider identifies search provider
type SearchProvider int

const (
    ProviderGoogle SearchProvider = iota
    ProviderBing
    ProviderDuckDuckGo
)

// TimeRange filters by time
type TimeRange string

const (
    TimeAny      TimeRange = ""
    TimeDay      TimeRange = "d"
    TimeWeek     TimeRange = "w"
    TimeMonth    TimeRange = "m"
    TimeYear     TimeRange = "y"
)

// SearchResult contains search results
type SearchResult struct {
    Query       string
    Provider    SearchProvider
    Results     []SearchItem
    TotalResults int64
    SearchTime  float64
    Timestamp   time.Time
}

// SearchItem represents a single result
type SearchItem struct {
    Title       string
    URL         string
    Snippet     string
    DisplayURL  string
    Position    int
    Favicon     string
    Metadata    map[string]interface{}
}
```

#### 3. Search Providers

```go
// Provider interface for search providers
type Provider interface {
    Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error)
    Name() string
}

// GoogleProvider implements Google search
type GoogleProvider struct {
    apiKey string
    cseID  string
    client *http.Client
}

// Search implements Provider
func (gp *GoogleProvider) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
    // Build request URL
    url := gp.buildSearchURL(query, opts)

    // Make request
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := gp.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("http request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("search failed: status %d", resp.StatusCode)
    }

    // Parse response
    var apiResp GoogleSearchResponse
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    // Convert to SearchResult
    return gp.convertResponse(&apiResp), nil
}

// buildSearchURL builds the search URL
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

// GoogleSearchResponse represents API response
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

// BingProvider implements Bing search
type BingProvider struct {
    apiKey string
    client *http.Client
}

// Search implements Provider
func (bp *BingProvider) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
    url := bp.buildSearchURL(query, opts)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    // Bing requires subscription key header
    req.Header.Set("Ocp-Apim-Subscription-Key", bp.apiKey)

    resp, err := bp.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("http request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("search failed: status %d", resp.StatusCode)
    }

    var apiResp BingSearchResponse
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    return bp.convertResponse(&apiResp), nil
}

// BingSearchResponse represents Bing API response
type BingSearchResponse struct {
    WebPages struct {
        Value []struct {
            Name        string `json:"name"`
            URL         string `json:"url"`
            Snippet     string `json:"snippet"`
            DisplayURL  string `json:"displayUrl"`
        } `json:"value"`
        TotalEstimatedMatches int64 `json:"totalEstimatedMatches"`
    } `json:"webPages"`
}

// DuckDuckGoProvider implements DuckDuckGo search
type DuckDuckGoProvider struct {
    client *http.Client
}

// Search implements Provider
func (ddg *DuckDuckGoProvider) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
    // DuckDuckGo HTML search (no official API)
    url := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    // Set user agent
    req.Header.Set("User-Agent", "Mozilla/5.0 (compatible)")

    resp, err := ddg.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("http request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("search failed: status %d", resp.StatusCode)
    }

    // Parse HTML
    return ddg.parseHTML(resp.Body, query)
}

// parseHTML parses DuckDuckGo HTML results
func (ddg *DuckDuckGoProvider) parseHTML(body io.Reader, query string) (*SearchResult, error) {
    doc, err := goquery.NewDocumentFromReader(body)
    if err != nil {
        return nil, err
    }

    result := &SearchResult{
        Query:     query,
        Provider:  ProviderDuckDuckGo,
        Results:   []SearchItem{},
        Timestamp: time.Now(),
    }

    // Extract results
    doc.Find(".result").Each(func(i int, s *goquery.Selection) {
        title := s.Find(".result__title").Text()
        url, _ := s.Find(".result__url").Attr("href")
        snippet := s.Find(".result__snippet").Text()

        if title != "" && url != "" {
            result.Results = append(result.Results, SearchItem{
                Title:    strings.TrimSpace(title),
                URL:      url,
                Snippet:  strings.TrimSpace(snippet),
                Position: i + 1,
            })
        }
    })

    return result, nil
}
```

#### 4. Fetcher

```go
// Fetcher fetches content from URLs
type Fetcher struct {
    client       *http.Client
    proxyManager *ProxyManager
    userAgents   []string
    rateLimiter  *RateLimiter
}

// Fetch fetches content from a URL
func (f *Fetcher) Fetch(ctx context.Context, url string, opts FetchOptions) (*FetchResult, error) {
    // Validate URL
    if err := validateURL(url); err != nil {
        return nil, fmt.Errorf("invalid url: %w", err)
    }

    // Check rate limit
    if err := f.rateLimiter.Wait(ctx, url); err != nil {
        return nil, fmt.Errorf("rate limit: %w", err)
    }

    // Build request
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    // Set headers
    f.setHeaders(req, opts)

    // Execute request
    resp, err := f.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("http request: %w", err)
    }
    defer resp.Body.Close()

    // Handle redirects
    if resp.StatusCode >= 300 && resp.StatusCode < 400 {
        location := resp.Header.Get("Location")
        return &FetchResult{
            URL:      url,
            Redirect: location,
            Status:   resp.StatusCode,
        }, nil
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("fetch failed: status %d", resp.StatusCode)
    }

    // Read body with size limit
    body, err := f.readBody(resp.Body, opts.MaxSize)
    if err != nil {
        return nil, fmt.Errorf("read body: %w", err)
    }

    return &FetchResult{
        URL:         url,
        Status:      resp.StatusCode,
        ContentType: resp.Header.Get("Content-Type"),
        Content:     body,
        Headers:     resp.Header,
        Size:        int64(len(body)),
        Timestamp:   time.Now(),
    }, nil
}

// FetchOptions configures fetch behavior
type FetchOptions struct {
    Headers   map[string]string
    Timeout   time.Duration
    MaxSize   int64
    UserAgent string
    Proxy     string
    FollowRedirects bool
}

// FetchResult contains fetched content
type FetchResult struct {
    URL         string
    Status      int
    ContentType string
    Content     []byte
    Headers     http.Header
    Size        int64
    Redirect    string
    Timestamp   time.Time
}

// setHeaders sets request headers
func (f *Fetcher) setHeaders(req *http.Request, opts FetchOptions) {
    // User agent
    userAgent := opts.UserAgent
    if userAgent == "" {
        userAgent = f.randomUserAgent()
    }
    req.Header.Set("User-Agent", userAgent)

    // Accept
    req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
    req.Header.Set("Accept-Language", "en-US,en;q=0.9")

    // Custom headers
    for k, v := range opts.Headers {
        req.Header.Set(k, v)
    }
}

// readBody reads response body with size limit
func (f *Fetcher) readBody(body io.Reader, maxSize int64) ([]byte, error) {
    if maxSize == 0 {
        maxSize = 10 * 1024 * 1024 // 10MB default
    }

    limited := io.LimitReader(body, maxSize)
    return io.ReadAll(limited)
}

// randomUserAgent returns a random user agent
func (f *Fetcher) randomUserAgent() string {
    if len(f.userAgents) == 0 {
        return "Mozilla/5.0 (compatible; HelixCode/1.0)"
    }
    return f.userAgents[rand.Intn(len(f.userAgents))]
}

// validateURL validates a URL
func validateURL(rawURL string) error {
    parsed, err := url.Parse(rawURL)
    if err != nil {
        return err
    }

    if parsed.Scheme != "http" && parsed.Scheme != "https" {
        return fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
    }

    if parsed.Host == "" {
        return fmt.Errorf("missing host")
    }

    return nil
}
```

#### 5. Parser

```go
// Parser converts HTML to markdown
type Parser struct {
    converter *md.Converter
}

// Parse converts HTML to markdown
func (p *Parser) Parse(html []byte, baseURL string) (*ParsedContent, error) {
    // Parse HTML
    doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
    if err != nil {
        return nil, fmt.Errorf("parse html: %w", err)
    }

    // Extract metadata
    metadata := p.extractMetadata(doc)

    // Remove unwanted elements
    p.cleanDocument(doc)

    // Convert to markdown
    htmlStr, _ := doc.Html()
    markdown := p.converter.ConvertString(htmlStr)

    // Clean markdown
    markdown = p.cleanMarkdown(markdown)

    return &ParsedContent{
        Markdown: markdown,
        Metadata: metadata,
        BaseURL:  baseURL,
    }, nil
}

// ParsedContent contains parsed content
type ParsedContent struct {
    Markdown string
    Metadata Metadata
    BaseURL  string
}

// Metadata contains document metadata
type Metadata struct {
    Title       string
    Description string
    Author      string
    Published   time.Time
    Modified    time.Time
    Keywords    []string
    Image       string
    Language    string
}

// extractMetadata extracts metadata from HTML
func (p *Parser) extractMetadata(doc *goquery.Document) Metadata {
    metadata := Metadata{}

    // Title
    metadata.Title = doc.Find("title").First().Text()
    if og := doc.Find("meta[property='og:title']").First(); og.Length() > 0 {
        if content, exists := og.Attr("content"); exists {
            metadata.Title = content
        }
    }

    // Description
    if desc := doc.Find("meta[name='description']").First(); desc.Length() > 0 {
        if content, exists := desc.Attr("content"); exists {
            metadata.Description = content
        }
    }

    // Author
    if author := doc.Find("meta[name='author']").First(); author.Length() > 0 {
        if content, exists := author.Attr("content"); exists {
            metadata.Author = content
        }
    }

    // Keywords
    if keywords := doc.Find("meta[name='keywords']").First(); keywords.Length() > 0 {
        if content, exists := keywords.Attr("content"); exists {
            metadata.Keywords = strings.Split(content, ",")
        }
    }

    // Image
    if og := doc.Find("meta[property='og:image']").First(); og.Length() > 0 {
        if content, exists := og.Attr("content"); exists {
            metadata.Image = content
        }
    }

    // Language
    if lang, exists := doc.Find("html").Attr("lang"); exists {
        metadata.Language = lang
    }

    return metadata
}

// cleanDocument removes unwanted elements
func (p *Parser) cleanDocument(doc *goquery.Document) {
    // Remove script and style tags
    doc.Find("script, style, noscript").Remove()

    // Remove navigation, header, footer
    doc.Find("nav, header, footer, aside").Remove()

    // Remove ads and social media
    doc.Find(".ad, .ads, .advertisement, .social-share").Remove()
    doc.Find("[class*='social'], [class*='share']").Remove()
}

// cleanMarkdown cleans up markdown output
func (p *Parser) cleanMarkdown(markdown string) string {
    // Remove excessive blank lines
    markdown = regexp.MustCompile(`\n{3,}`).ReplaceAllString(markdown, "\n\n")

    // Trim whitespace
    lines := strings.Split(markdown, "\n")
    for i, line := range lines {
        lines[i] = strings.TrimRight(line, " \t")
    }
    markdown = strings.Join(lines, "\n")

    return strings.TrimSpace(markdown)
}
```

#### 6. CacheManager

```go
// CacheManager manages caching of web content
type CacheManager struct {
    memCache  *sync.Map
    diskCache *DiskCache
    ttl       time.Duration
}

// NewCacheManager creates a new cache manager
func NewCacheManager(cacheDir string, ttl time.Duration) *CacheManager {
    return &CacheManager{
        memCache:  &sync.Map{},
        diskCache: NewDiskCache(cacheDir),
        ttl:       ttl,
    }
}

// GetSearch retrieves cached search result
func (cm *CacheManager) GetSearch(query string) (*SearchResult, bool) {
    return cm.get("search:" + query)
}

// SetSearch caches search result
func (cm *CacheManager) SetSearch(query string, result *SearchResult) {
    cm.set("search:"+query, result)
}

// GetFetch retrieves cached fetch result
func (cm *CacheManager) GetFetch(url string) (*FetchResult, bool) {
    return cm.get("fetch:" + url)
}

// SetFetch caches fetch result
func (cm *CacheManager) SetFetch(url string, result *FetchResult) {
    cm.set("fetch:"+url, result)
}

// get retrieves from cache
func (cm *CacheManager) get(key string) (interface{}, bool) {
    // Check memory cache
    if val, ok := cm.memCache.Load(key); ok {
        entry := val.(*CacheEntry)
        if time.Since(entry.Timestamp) < cm.ttl {
            return entry.Value, true
        }
        // Expired
        cm.memCache.Delete(key)
    }

    // Check disk cache
    if val, ok := cm.diskCache.Get(key); ok {
        entry := val.(*CacheEntry)
        if time.Since(entry.Timestamp) < cm.ttl {
            // Restore to memory cache
            cm.memCache.Store(key, entry)
            return entry.Value, true
        }
    }

    return nil, false
}

// set stores in cache
func (cm *CacheManager) set(key string, value interface{}) {
    entry := &CacheEntry{
        Value:     value,
        Timestamp: time.Now(),
    }

    // Store in memory
    cm.memCache.Store(key, entry)

    // Store on disk
    cm.diskCache.Set(key, entry)
}

// CacheEntry represents a cached item
type CacheEntry struct {
    Value     interface{}
    Timestamp time.Time
}

// DiskCache manages disk-based caching
type DiskCache struct {
    dir string
    mu  sync.RWMutex
}

// Get retrieves from disk cache
func (dc *DiskCache) Get(key string) (*CacheEntry, bool) {
    dc.mu.RLock()
    defer dc.mu.RUnlock()

    path := dc.keyPath(key)
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, false
    }

    var entry CacheEntry
    if err := json.Unmarshal(data, &entry); err != nil {
        return nil, false
    }

    return &entry, true
}

// Set stores in disk cache
func (dc *DiskCache) Set(key string, entry *CacheEntry) error {
    dc.mu.Lock()
    defer dc.mu.Unlock()

    path := dc.keyPath(key)

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return err
    }

    data, err := json.Marshal(entry)
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0644)
}

// keyPath converts key to file path
func (dc *DiskCache) keyPath(key string) string {
    hash := sha256.Sum256([]byte(key))
    hashStr := fmt.Sprintf("%x", hash)
    return filepath.Join(dc.dir, hashStr[:2], hashStr[2:4], hashStr)
}

// Cleanup removes expired entries
func (dc *DiskCache) Cleanup(ttl time.Duration) error {
    return filepath.Walk(dc.dir, func(path string, info os.FileInfo, err error) error {
        if err != nil || info.IsDir() {
            return err
        }

        if time.Since(info.ModTime()) > ttl {
            return os.Remove(path)
        }

        return nil
    })
}
```

#### 7. RateLimiter

```go
// RateLimiter manages rate limiting
type RateLimiter struct {
    mu       sync.RWMutex
    limiters map[string]*rate.Limiter
    limits   map[string]RateLimit
}

// RateLimit defines rate limit configuration
type RateLimit struct {
    RequestsPerSecond float64
    Burst             int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
    return &RateLimiter{
        limiters: make(map[string]*rate.Limiter),
        limits: map[string]RateLimit{
            "google":     {RequestsPerSecond: 10, Burst: 20},
            "bing":       {RequestsPerSecond: 5, Burst: 10},
            "duckduckgo": {RequestsPerSecond: 2, Burst: 5},
            "default":    {RequestsPerSecond: 1, Burst: 3},
        },
    }
}

// Wait waits for rate limit
func (rl *RateLimiter) Wait(ctx context.Context, key string) error {
    limiter := rl.getLimiter(key)
    return limiter.Wait(ctx)
}

// getLimiter gets or creates limiter for key
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
    rl.mu.RLock()
    limiter, ok := rl.limiters[key]
    rl.mu.RUnlock()

    if ok {
        return limiter
    }

    rl.mu.Lock()
    defer rl.mu.Unlock()

    // Double-check
    if limiter, ok := rl.limiters[key]; ok {
        return limiter
    }

    // Create new limiter
    limit := rl.limits[key]
    if limit.RequestsPerSecond == 0 {
        limit = rl.limits["default"]
    }

    limiter = rate.NewLimiter(rate.Limit(limit.RequestsPerSecond), limit.Burst)
    rl.limiters[key] = limiter

    return limiter
}

// SetLimit sets rate limit for a key
func (rl *RateLimiter) SetLimit(key string, limit RateLimit) {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    rl.limits[key] = limit
    delete(rl.limiters, key) // Will be recreated with new limit
}
```

#### 8. ProxyManager

```go
// ProxyManager manages proxy rotation
type ProxyManager struct {
    proxies []string
    current int
    mu      sync.Mutex
}

// NewProxyManager creates a proxy manager
func NewProxyManager(proxies []string) *ProxyManager {
    return &ProxyManager{
        proxies: proxies,
    }
}

// GetProxy gets next proxy
func (pm *ProxyManager) GetProxy() string {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    if len(pm.proxies) == 0 {
        return ""
    }

    proxy := pm.proxies[pm.current]
    pm.current = (pm.current + 1) % len(pm.proxies)

    return proxy
}

// GetHTTPTransport creates HTTP transport with proxy
func (pm *ProxyManager) GetHTTPTransport() *http.Transport {
    proxyURL := pm.GetProxy()
    if proxyURL == "" {
        return &http.Transport{}
    }

    proxy, err := url.Parse(proxyURL)
    if err != nil {
        return &http.Transport{}
    }

    return &http.Transport{
        Proxy: http.ProxyURL(proxy),
    }
}
```

## Configuration Schema

```yaml
# web_tools.yaml

web_tools:
  # Search configuration
  search:
    # Default provider
    default_provider: google  # google, bing, duckduckgo

    # Provider configurations
    providers:
      google:
        enabled: true
        api_key: ${GOOGLE_API_KEY}
        cse_id: ${GOOGLE_CSE_ID}
        rate_limit:
          requests_per_second: 10
          burst: 20

      bing:
        enabled: true
        api_key: ${BING_API_KEY}
        rate_limit:
          requests_per_second: 5
          burst: 10

      duckduckgo:
        enabled: true
        rate_limit:
          requests_per_second: 2
          burst: 5

    # Search options
    options:
      max_results: 10
      safe_search: true
      timeout: 30s

  # Fetch configuration
  fetch:
    # HTTP client
    timeout: 30s
    max_size: 10MB
    follow_redirects: true
    max_redirects: 10

    # User agents for rotation
    user_agents:
      - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
      - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
      - "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"

  # Parsing configuration
  parsing:
    # HTML to markdown
    remove_scripts: true
    remove_styles: true
    remove_navigation: true
    extract_metadata: true

  # Caching
  cache:
    enabled: true
    ttl: 15m
    dir: .helix/cache/web
    max_size: 1GB

    # Memory cache
    memory_cache_size: 100MB

    # Cleanup
    cleanup_interval: 1h

  # Rate limiting
  rate_limit:
    enabled: true

    # Per-domain limits
    per_domain:
      requests_per_second: 1
      burst: 3

  # Proxy configuration
  proxy:
    enabled: false
    rotation: true
    proxies:
      - http://proxy1.example.com:8080
      - http://proxy2.example.com:8080

  # Security
  security:
    # Blocked domains
    blocked_domains:
      - "*.onion"
      - "*.i2p"

    # Max content size
    max_content_size: 10MB

    # Timeout
    max_timeout: 60s
```

```go
// Config represents web tools configuration
type Config struct {
    Search  SearchConfig  `yaml:"search"`
    Fetch   FetchConfig   `yaml:"fetch"`
    Parsing ParsingConfig `yaml:"parsing"`
    Cache   CacheConfig   `yaml:"cache"`
    RateLimit RateLimitConfig `yaml:"rate_limit"`
    Proxy   ProxyConfig   `yaml:"proxy"`
    Security SecurityConfig `yaml:"security"`
}

// SearchConfig configures search
type SearchConfig struct {
    DefaultProvider SearchProvider           `yaml:"default_provider"`
    Providers       map[string]ProviderConfig `yaml:"providers"`
    Options         SearchOptionsConfig      `yaml:"options"`
}

// ProviderConfig configures a provider
type ProviderConfig struct {
    Enabled   bool       `yaml:"enabled"`
    APIKey    string     `yaml:"api_key"`
    CSEID     string     `yaml:"cse_id"`
    RateLimit RateLimit  `yaml:"rate_limit"`
}
```

## Testing Strategy

### Unit Tests

```go
func TestSearchEngine_Search(t *testing.T) {
    mockProvider := &MockProvider{
        results: &SearchResult{
            Query: "test query",
            Results: []SearchItem{
                {
                    Title:   "Test Result",
                    URL:     "https://example.com",
                    Snippet: "Test snippet",
                },
            },
        },
    }

    se := NewSearchEngine()
    se.providers[ProviderGoogle] = mockProvider

    result, err := se.Search(context.Background(), "test query", SearchOptions{
        Provider: ProviderGoogle,
    })

    require.NoError(t, err)
    assert.Len(t, result.Results, 1)
    assert.Equal(t, "Test Result", result.Results[0].Title)
}

func TestFetcher_Fetch(t *testing.T) {
    // Create test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        w.Write([]byte("<html><body>Test content</body></html>"))
    }))
    defer server.Close()

    fetcher := NewFetcher()

    result, err := fetcher.Fetch(context.Background(), server.URL, FetchOptions{})
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, result.Status)
    assert.Contains(t, string(result.Content), "Test content")
}

func TestParser_Parse(t *testing.T) {
    parser := NewParser()

    html := []byte(`
        <html>
        <head>
            <title>Test Page</title>
            <meta name="description" content="Test description">
        </head>
        <body>
            <h1>Hello World</h1>
            <p>This is a test.</p>
        </body>
        </html>
    `)

    parsed, err := parser.Parse(html, "https://example.com")
    require.NoError(t, err)
    assert.Equal(t, "Test Page", parsed.Metadata.Title)
    assert.Contains(t, parsed.Markdown, "Hello World")
}

func TestCacheManager_GetSet(t *testing.T) {
    tmpDir := t.TempDir()
    cm := NewCacheManager(tmpDir, 15*time.Minute)

    // Set
    result := &SearchResult{
        Query: "test",
    }
    cm.SetSearch("test", result)

    // Get
    cached, ok := cm.GetSearch("test")
    assert.True(t, ok)
    assert.Equal(t, "test", cached.Query)

    // Get non-existent
    _, ok = cm.GetSearch("nonexistent")
    assert.False(t, ok)
}

func TestRateLimiter_Wait(t *testing.T) {
    rl := NewRateLimiter()
    rl.SetLimit("test", RateLimit{
        RequestsPerSecond: 10,
        Burst:             1,
    })

    ctx := context.Background()

    // First request should succeed immediately
    err := rl.Wait(ctx, "test")
    assert.NoError(t, err)

    // Second request should wait
    start := time.Now()
    err = rl.Wait(ctx, "test")
    assert.NoError(t, err)
    duration := time.Since(start)
    assert.Greater(t, duration, 50*time.Millisecond)
}
```

### Integration Tests

```go
func TestWebToolsCoordinator_EndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    coordinator := NewWebToolsCoordinator()

    // Test search
    searchResult, err := coordinator.Search(context.Background(), "golang", SearchOptions{
        Provider:   ProviderDuckDuckGo,
        MaxResults: 5,
    })
    require.NoError(t, err)
    assert.Greater(t, len(searchResult.Results), 0)

    // Test fetch and parse
    if len(searchResult.Results) > 0 {
        url := searchResult.Results[0].URL
        markdown, err := coordinator.FetchAndParse(context.Background(), url)
        require.NoError(t, err)
        assert.NotEmpty(t, markdown)
    }
}

func TestCaching(t *testing.T) {
    tmpDir := t.TempDir()
    coordinator := NewWebToolsCoordinator(
        WithCacheDir(tmpDir),
        WithCacheTTL(1*time.Minute),
    )

    query := "test query"

    // First search - should hit API
    start1 := time.Now()
    _, err := coordinator.Search(context.Background(), query, SearchOptions{})
    require.NoError(t, err)
    duration1 := time.Since(start1)

    // Second search - should hit cache
    start2 := time.Now()
    _, err = coordinator.Search(context.Background(), query, SearchOptions{})
    require.NoError(t, err)
    duration2 := time.Since(start2)

    // Cached request should be much faster
    assert.Less(t, duration2, duration1/10)
}
```

## Performance Considerations

### Connection Pooling

```go
// Configure HTTP client with connection pooling
func newHTTPClient() *http.Client {
    return &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
            DisableCompression:  false,
        },
    }
}
```

### Parallel Fetching

```go
// FetchMultiple fetches multiple URLs in parallel
func (wtc *WebToolsCoordinator) FetchMultiple(ctx context.Context, urls []string) ([]*FetchResult, error) {
    results := make([]*FetchResult, len(urls))
    errors := make([]error, len(urls))

    var wg sync.WaitGroup
    semaphore := make(chan struct{}, 5) // Limit concurrency

    for i, url := range urls {
        wg.Add(1)
        go func(idx int, u string) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            result, err := wtc.Fetch(ctx, u, FetchOptions{})
            results[idx] = result
            errors[idx] = err
        }(i, url)
    }

    wg.Wait()

    // Check for errors
    for _, err := range errors {
        if err != nil {
            return results, err
        }
    }

    return results, nil
}
```

## Security Considerations

### URL Validation

```go
// ValidateURL validates and sanitizes URLs
func ValidateURL(rawURL string) error {
    parsed, err := url.Parse(rawURL)
    if err != nil {
        return fmt.Errorf("invalid url: %w", err)
    }

    // Check scheme
    if parsed.Scheme != "http" && parsed.Scheme != "https" {
        return fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
    }

    // Check for local/private IPs
    if isPrivateIP(parsed.Host) {
        return fmt.Errorf("private IP not allowed: %s", parsed.Host)
    }

    // Check blocked domains
    if isBlockedDomain(parsed.Host) {
        return fmt.Errorf("blocked domain: %s", parsed.Host)
    }

    return nil
}

func isPrivateIP(host string) bool {
    ip := net.ParseIP(host)
    if ip == nil {
        return false
    }

    return ip.IsPrivate() || ip.IsLoopback()
}
```

### Content Sanitization

```go
// SanitizeHTML removes potentially dangerous HTML
func SanitizeHTML(html []byte) []byte {
    policy := bluemonday.UGCPolicy()
    return policy.SanitizeBytes(html)
}
```

## References

### Qwen Code Web Tools

- **Feature**: Web search and fetch capabilities
- **Implementation**: Multiple provider support
- **Caching**: In-memory and disk caching

### Cline Web Fetch

- **Repository**: `src/tools/web-fetch.ts`
- **Features**:
  - URL fetching
  - HTML to markdown
  - Redirect handling

### Key Insights

1. **Multiple Providers**: Support multiple search engines for redundancy
2. **Caching**: Essential for performance and rate limit management
3. **Rate Limiting**: Respect API limits and avoid bans
4. **Clean Output**: Convert HTML to clean markdown for LLMs
5. **Security**: Validate URLs and sanitize content

## Usage Examples

```go
// Example 1: Search
func ExampleSearch() {
    coordinator := NewWebToolsCoordinator()

    result, _ := coordinator.Search(context.Background(), "golang best practices", SearchOptions{
        Provider:   ProviderGoogle,
        MaxResults: 10,
    })

    for _, item := range result.Results {
        fmt.Printf("%s - %s\n", item.Title, item.URL)
    }
}

// Example 2: Fetch and parse
func ExampleFetchAndParse() {
    coordinator := NewWebToolsCoordinator()

    markdown, _ := coordinator.FetchAndParse(
        context.Background(),
        "https://golang.org/doc/",
    )

    fmt.Println(markdown)
}

// Example 3: With proxy
func ExampleWithProxy() {
    coordinator := NewWebToolsCoordinator(
        WithProxies([]string{
            "http://proxy.example.com:8080",
        }),
    )

    result, _ := coordinator.Fetch(context.Background(), "https://example.com", FetchOptions{})
    fmt.Println(string(result.Content))
}
```

## Future Enhancements

1. **JavaScript Rendering**: Support for dynamic content (Playwright/Puppeteer)
2. **PDF Parsing**: Extract text from PDF documents
3. **Image OCR**: Extract text from images
4. **Video Transcription**: Extract text from videos
5. **Advanced Parsing**: Better content extraction algorithms
