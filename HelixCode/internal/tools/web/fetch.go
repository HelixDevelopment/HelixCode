package web

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Fetcher fetches content from URLs
type Fetcher struct {
	httpClient   *http.Client
	cacheManager *CacheManager
	rateLimiter  *RateLimiter
	config       *Config
}

// FetchOptions configures fetch behavior
type FetchOptions struct {
	Headers         map[string]string
	Timeout         time.Duration
	MaxSize         int64
	UserAgent       string
	FollowRedirects bool
	ValidateSSL     bool
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

// NewFetcher creates a new fetcher
func NewFetcher(httpClient *http.Client, cacheManager *CacheManager, rateLimiter *RateLimiter, config *Config) *Fetcher {
	return &Fetcher{
		httpClient:   httpClient,
		cacheManager: cacheManager,
		rateLimiter:  rateLimiter,
		config:       config,
	}
}

// Fetch fetches content from a URL
func (f *Fetcher) Fetch(ctx context.Context, rawURL string, opts FetchOptions) (*FetchResult, error) {
	// Validate URL
	if err := f.validateURL(rawURL); err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	// Check cache
	if f.cacheManager != nil {
		cacheKey := fmt.Sprintf("fetch:%s", rawURL)
		if cached, ok := f.cacheManager.Get(cacheKey); ok {
			return &FetchResult{
				URL:         rawURL,
				Status:      http.StatusOK,
				Content:     cached,
				Size:        int64(len(cached)),
				Timestamp:   time.Now(),
				ContentType: "text/html",
			}, nil
		}
	}

	// Check rate limit
	if f.rateLimiter != nil {
		parsed, _ := url.Parse(rawURL)
		if err := f.rateLimiter.Wait(ctx, parsed.Host); err != nil {
			return nil, fmt.Errorf("rate limit: %w", err)
		}
	}

	// Build request
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	f.setHeaders(req, opts)

	// Execute request
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// Handle redirects
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		return &FetchResult{
			URL:      rawURL,
			Redirect: location,
			Status:   resp.StatusCode,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("fetch failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Read body with size limit
	maxSize := opts.MaxSize
	if maxSize == 0 {
		maxSize = f.config.MaxContentSize
	}

	body, err := f.readBody(resp.Body, maxSize)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	result := &FetchResult{
		URL:         rawURL,
		Status:      resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Content:     body,
		Headers:     resp.Header,
		Size:        int64(len(body)),
		Timestamp:   time.Now(),
	}

	// Cache result
	if f.cacheManager != nil {
		cacheKey := fmt.Sprintf("fetch:%s", rawURL)
		f.cacheManager.Set(cacheKey, body)
	}

	return result, nil
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
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	// Custom headers
	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}
}

// readBody reads response body with size limit
func (f *Fetcher) readBody(body io.Reader, maxSize int64) ([]byte, error) {
	limited := io.LimitReader(body, maxSize)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return data, nil
}

// randomUserAgent returns a random user agent
func (f *Fetcher) randomUserAgent() string {
	if len(f.config.UserAgents) == 0 {
		return "Mozilla/5.0 (compatible; HelixCode/1.0)"
	}
	return f.config.UserAgents[rand.Intn(len(f.config.UserAgents))]
}

// validateURL validates a URL
func (f *Fetcher) validateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	// Check scheme
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
	}

	// Check host
	if parsed.Host == "" {
		return fmt.Errorf("missing host")
	}

	// Check blocked domains
	for _, blocked := range f.config.BlockedDomains {
		if matchDomain(parsed.Host, blocked) {
			return fmt.Errorf("blocked domain: %s", parsed.Host)
		}
	}

	// Check private IPs
	if !f.config.AllowPrivateIPs {
		if isPrivateIP(parsed.Host) {
			return fmt.Errorf("private IP not allowed: %s", parsed.Host)
		}
	}

	return nil
}

// matchDomain checks if a domain matches a pattern
func matchDomain(domain, pattern string) bool {
	// Simple wildcard matching
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		return strings.HasSuffix(domain, suffix) || domain == suffix[1:]
	}
	return domain == pattern
}

// isPrivateIP checks if a host is a private IP
func isPrivateIP(host string) bool {
	// Extract host without port
	if strings.Contains(host, ":") {
		var err error
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			return false
		}
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Try to resolve hostname
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return false
		}
		ip = ips[0]
	}

	return ip.IsPrivate() || ip.IsLoopback()
}

// FetchMultiple fetches multiple URLs concurrently
func (f *Fetcher) FetchMultiple(ctx context.Context, urls []string, opts FetchOptions) ([]*FetchResult, error) {
	results := make([]*FetchResult, len(urls))
	errors := make([]error, len(urls))

	// Semaphore to limit concurrency
	semaphore := make(chan struct{}, 5)

	// Wait group for all fetches
	done := make(chan struct{})
	go func() {
		for i, url := range urls {
			select {
			case <-ctx.Done():
				errors[i] = ctx.Err()
				continue
			case semaphore <- struct{}{}:
			}

			go func(idx int, u string) {
				defer func() { <-semaphore }()
				result, err := f.Fetch(ctx, u, opts)
				results[idx] = result
				errors[idx] = err
			}(i, url)
		}

		// Wait for all goroutines to finish
		for i := 0; i < cap(semaphore); i++ {
			semaphore <- struct{}{}
		}
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return results, err
		}
	}

	return results, nil
}
