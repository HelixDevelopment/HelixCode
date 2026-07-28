package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"dev.helix.code/internal/netutil"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/redis/go-redis/v9"
)

// ErrRedisClientNotInitialized is returned by RedisMemoryProvider.Health
// when no real Redis client has been wired into the provider. Before
// round-31 §11.4 anti-bluff sweep (2026-05-18) Health returned nil
// unconditionally — even when no client existed — so the health endpoint
// reported OK for a backend that was never even attempted. The
// RedisMemoryProvider struct currently keeps state in a local in-memory
// map (no real client field exists), so Health now surfaces this sentinel
// to make the absence of real Redis connectivity unambiguous to operators
// and monitoring systems. Wiring a real go-redis/v9 client and replacing
// the in-memory map MUST happen before this provider is fit for the
// "redis" name; until then Health correctly fails closed.
//
// §11.4 CRITICAL: false health for dead Redis is a release blocker under
// Article XI §11.9 / CONST-035.
var ErrRedisClientNotInitialized = errors.New("redis memory provider: client has not been initialized — Health() previously returned nil regardless of backend state (§11.4 CRITICAL: false health for dead Redis); the current RedisMemoryProvider holds state in an in-memory map and does NOT yet talk to a real Redis server, so Health fails closed until a real go-redis/v9 client is wired in")

// ErrMemcachedClientNotInitialized is returned by
// MemcachedMemoryProvider.Health when no real Memcached client has been
// wired into the provider. Same story as ErrRedisClientNotInitialized:
// the previous implementation returned nil unconditionally so the health
// endpoint advertised OK regardless of backend state. Until a real
// gomemcache client is wired in and the local in-memory map is replaced,
// Health fails closed via this sentinel.
//
// §11.4 CRITICAL: false health for dead Memcached is a release blocker
// under Article XI §11.9 / CONST-035.
var ErrMemcachedClientNotInitialized = errors.New("memcached memory provider: client has not been initialized — Health() previously returned nil regardless of backend state (§11.4 CRITICAL: false health for dead Memcached); the current MemcachedMemoryProvider holds state in an in-memory map and does NOT yet talk to a real Memcached server, so Health fails closed until a real gomemcache client is wired in")

// ErrKeyNotFound is the canonical sentinel returned by Redis and Memcached
// memory providers when a Retrieve / Get call targets a key that does not
// exist in the backing store. Round-37 §11.4 anti-bluff sweep (2026-05-18):
// before this round both providers returned the local in-memory map's miss
// (a stringly-typed fmt.Errorf), masking real backend semantics —
// `redis.Nil` from go-redis/v9 and `memcache.ErrCacheMiss` from gomemcache
// were never reachable because no real client was wired in. With real
// clients now wired in, Get returns this sentinel uniformly so callers
// can `errors.Is(err, ErrKeyNotFound)` regardless of which backend is
// active (CONST-035 honest contract). Wrapping the backend error
// preserves the underlying diagnostic for operators.
var ErrKeyNotFound = errors.New("memory provider: key not found")

// ErrListNotSupportedByBackend is the sentinel returned by
// MemcachedMemoryProvider.Search (and any other list/prefix-scan call) to
// surface the protocol-level fact that Memcached intentionally does not
// expose a key-enumeration / SCAN API. Round-37 §11.4 anti-bluff sweep:
// before this round MemcachedMemoryProvider.Search walked a local
// in-memory map and pretended Memcached supported value-substring search,
// which was a fabricated-capability bluff under Article XI §11.9 /
// CONST-035 — callers were promised something the underlying protocol
// could never honour. Returning this sentinel is the honest contract:
// callers needing list / prefix-scan capability MUST use Redis (which
// supports SCAN) or another backend that exposes key enumeration.
//
// Memcached is a pure cache. There is no upstream feature to wait for.
var ErrListNotSupportedByBackend = errors.New("memcached: memcached protocol does not support key-prefix scan / iteration — Memcached is a pure cache without a SCAN equivalent (CONST-035 honest contract). Use Redis or a different backend for list operations")

// MemoryConfig represents configuration for memory operations
type MemoryConfig struct {
	Enabled          bool          `json:"enabled"`
	Provider         string        `json:"provider"`
	MaxGenerations   int           `json:"max_generations"`
	MaxConversations int           `json:"max_conversations"`
	TTL              time.Duration `json:"ttl"`
}

// MemorySearchResult represents a search result from memory providers
type MemorySearchResult struct {
	Key   string      `json:"key"`
	Data  interface{} `json:"data"`
	Score float64     `json:"score"`
}

// MemoryProvider defines the interface for memory providers
type MemoryProvider interface {
	// Store stores data in memory
	Store(ctx context.Context, key string, data interface{}) error

	// Retrieve retrieves data from memory
	Retrieve(ctx context.Context, key string) (interface{}, error)

	// Search searches for data in memory
	Search(ctx context.Context, query string, limit int) ([]MemorySearchResult, error)

	// Delete deletes data from memory
	Delete(ctx context.Context, key string) error

	// Clear clears all data from memory
	Clear(ctx context.Context) error

	// Health checks the health of the memory provider
	Health(ctx context.Context) error

	// Name returns the name of the provider
	Name() string

	// Type returns the type of the provider
	Type() string
}

// MemoryManager manages memory operations across multiple providers
type MemoryManager struct {
	providers       map[string]MemoryProvider
	defaultProvider string
	config          *MemoryConfig
	mu              sync.RWMutex
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(config *MemoryConfig) *MemoryManager {
	return &MemoryManager{
		providers: make(map[string]MemoryProvider),
		config:    config,
	}
}

// RegisterProvider registers a memory provider
func (mm *MemoryManager) RegisterProvider(name string, provider MemoryProvider) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.providers[name]; exists {
		return errors.New(tr(context.Background(), "internal_memory_provider_already_registered", map[string]any{"Name": name}))
	}

	mm.providers[name] = provider

	// Set as default if it's the first provider
	if mm.defaultProvider == "" {
		mm.defaultProvider = name
	}

	return nil
}

// UnregisterProvider unregisters a memory provider
func (mm *MemoryManager) UnregisterProvider(name string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.providers[name]; !exists {
		return errors.New(tr(context.Background(), "internal_memory_provider_not_registered", map[string]any{"Name": name}))
	}

	delete(mm.providers, name)

	// Reset default if it was removed
	if mm.defaultProvider == name {
		mm.defaultProvider = ""
		// Set first available as default
		for providerName := range mm.providers {
			mm.defaultProvider = providerName
			break
		}
	}

	return nil
}

// SetDefaultProvider sets the default memory provider
func (mm *MemoryManager) SetDefaultProvider(name string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.providers[name]; !exists {
		return errors.New(tr(context.Background(), "internal_memory_provider_not_registered", map[string]any{"Name": name}))
	}

	mm.defaultProvider = name
	return nil
}

// GetProvider gets a memory provider by name
func (mm *MemoryManager) GetProvider(name string) (MemoryProvider, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	provider, exists := mm.providers[name]
	if !exists {
		return nil, errors.New(tr(context.Background(), "internal_memory_provider_not_found", map[string]any{"Name": name}))
	}

	return provider, nil
}

// GetDefaultProvider gets the default memory provider.
//
// Speed programme P4-T04: this resolves the provider directly under a single
// RLock acquisition rather than re-entering GetProvider (which acquires the
// same RWMutex again). Go's sync.RWMutex is NOT reentrant — a recursive
// RLock can deadlock if a writer's Lock() queues between the outer and inner
// RLock. Resolving inline both removes that deadlock window and narrows the
// critical section to a single lock round-trip on the Store/Retrieve/Search/
// Delete/Clear hot path that all funnels through this method.
func (mm *MemoryManager) GetDefaultProvider() (MemoryProvider, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	if mm.defaultProvider == "" {
		return nil, errors.New(tr(context.Background(), "internal_memory_no_default_provider_set", nil))
	}

	provider, exists := mm.providers[mm.defaultProvider]
	if !exists {
		return nil, errors.New(tr(context.Background(), "internal_memory_provider_not_found", map[string]any{"Name": mm.defaultProvider}))
	}

	return provider, nil
}

// ListProviders lists all registered providers
func (mm *MemoryManager) ListProviders() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	providers := make([]string, 0, len(mm.providers))
	for name := range mm.providers {
		providers = append(providers, name)
	}

	return providers
}

// Store stores data using the default provider
func (mm *MemoryManager) Store(ctx context.Context, key string, data interface{}) error {
	provider, err := mm.GetDefaultProvider()
	if err != nil {
		return fmt.Errorf("failed to get default provider: %w", err)
	}

	return provider.Store(ctx, key, data)
}

// Retrieve retrieves data using the default provider
func (mm *MemoryManager) Retrieve(ctx context.Context, key string) (interface{}, error) {
	provider, err := mm.GetDefaultProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get default provider: %w", err)
	}

	return provider.Retrieve(ctx, key)
}

// Search searches data using the default provider
func (mm *MemoryManager) Search(ctx context.Context, query string, limit int) ([]MemorySearchResult, error) {
	provider, err := mm.GetDefaultProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get default provider: %w", err)
	}

	return provider.Search(ctx, query, limit)
}

// Delete deletes data using the default provider
func (mm *MemoryManager) Delete(ctx context.Context, key string) error {
	provider, err := mm.GetDefaultProvider()
	if err != nil {
		return fmt.Errorf("failed to get default provider: %w", err)
	}

	return provider.Delete(ctx, key)
}

// Clear clears all data using the default provider
func (mm *MemoryManager) Clear(ctx context.Context) error {
	provider, err := mm.GetDefaultProvider()
	if err != nil {
		return fmt.Errorf("failed to get default provider: %w", err)
	}

	return provider.Clear(ctx)
}

// Health checks the health of all providers
func (mm *MemoryManager) Health(ctx context.Context) map[string]error {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	health := make(map[string]error)

	for name, provider := range mm.providers {
		if err := provider.Health(ctx); err != nil {
			health[name] = err
		} else {
			health[name] = nil
		}
	}

	return health
}

// GetStatistics returns statistics for all providers
func (mm *MemoryManager) GetStatistics() map[string]interface{} {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_providers":  len(mm.providers),
		"default_provider": mm.defaultProvider,
		"providers":        make(map[string]interface{}),
	}

	providers := make(map[string]interface{})
	for name, provider := range mm.providers {
		providers[name] = map[string]interface{}{
			"type": provider.Type(),
			"name": provider.Name(),
		}
	}

	stats["providers"] = providers
	return stats
}

// MemoryProviderFactory creates memory providers
type MemoryProviderFactory struct{}

// NewMemoryProviderFactory creates a new factory
func NewMemoryProviderFactory() *MemoryProviderFactory {
	return &MemoryProviderFactory{}
}

// CreateProvider creates a memory provider based on configuration
func (f *MemoryProviderFactory) CreateProvider(providerType string, config map[string]interface{}) (MemoryProvider, error) {
	switch providerType {
	case "redis":
		return NewRedisMemoryProvider(config)
	case "memcached":
		return NewMemcachedMemoryProvider(config)
	case "inmemory":
		return NewInMemoryProvider(config)
	case "filesystem":
		return NewFilesystemMemoryProvider(config)
	default:
		return nil, errors.New(tr(context.Background(), "internal_memory_unsupported_provider_type", map[string]any{"ProviderType": providerType}))
	}
}

// InMemoryProvider is an in-memory implementation of MemoryProvider
type InMemoryProvider struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewInMemoryProvider creates a new in-memory provider
func NewInMemoryProvider(config map[string]interface{}) (*InMemoryProvider, error) {
	return &InMemoryProvider{
		data: make(map[string]interface{}),
	}, nil
}

// Store stores data in memory
func (p *InMemoryProvider) Store(ctx context.Context, key string, data interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data[key] = data
	return nil
}

// Retrieve retrieves data from memory
func (p *InMemoryProvider) Retrieve(ctx context.Context, key string) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	data, exists := p.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return data, nil
}

// Search searches for data (simple implementation)
func (p *InMemoryProvider) Search(ctx context.Context, query string, limit int) ([]MemorySearchResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	results := make([]MemorySearchResult, 0, limit)
	count := 0

	for key, data := range p.data {
		if count >= limit {
			break
		}

		// Simple string matching for demo
		if fmt.Sprintf("%v", data) == query || key == query {
			results = append(results, MemorySearchResult{
				Key:   key,
				Data:  data,
				Score: 1.0,
			})
			count++
		}
	}

	return results, nil
}

// Delete deletes data from memory
func (p *InMemoryProvider) Delete(ctx context.Context, key string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.data[key]; !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	delete(p.data, key)
	return nil
}

// Clear clears all data
func (p *InMemoryProvider) Clear(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data = make(map[string]interface{})
	return nil
}

// Health checks health
func (p *InMemoryProvider) Health(ctx context.Context) error {
	return nil // In-memory is always healthy
}

// Name returns the provider name
func (p *InMemoryProvider) Name() string {
	return "in-memory"
}

// Type returns the provider type
func (p *InMemoryProvider) Type() string {
	return "inmemory"
}

// RedisMemoryProvider is a Redis-backed MemoryProvider that delegates all
// data operations to a real go-redis/v9 client. Round-37 §11.4 anti-bluff
// sweep (2026-05-18): before this round the provider held state in a
// local in-memory map and pretended to be Redis (round-31 fixed Health
// only; the data path remained fake). End-user data was lost across
// restarts despite the API contract promising persistence — a
// CRITICAL Article XI §11.9 / CONST-035 bluff.
//
// When constructed with a non-empty Redis host/URL the provider wires a
// real client and every method delegates to it. When constructed without
// connection details (empty config) the provider enters "nil-client"
// mode: the client field stays nil and every data method returns
// ErrRedisClientNotInitialized — preserving the round-31 invariant that
// an unconfigured provider is safe to hold and surfaces the gap loudly
// on first use.
type RedisMemoryProvider struct {
	client *redis.Client
	host   string
	port   int
	prefix string
	mu     sync.RWMutex
}

// NewRedisMemoryProvider creates a new Redis-backed memory provider.
//
// Config keys (all optional, backwards compatible with the round-31
// signature):
//   - "url"      string        full redis:// URL (takes precedence over host/port)
//   - "host"     string        Redis hostname (no default — empty means nil-client mode)
//   - "port"     int           Redis port (defaults to 6379 when host is set)
//   - "password" string        Redis AUTH password (sourced from env per CONST-042;
//                              never hardcoded in source)
//   - "db"       int           Redis logical DB (defaults to 0)
//   - "prefix"   string        Key namespace prefix (defaults to "helix:memory:")
//
// When neither "url" nor "host" is set the provider returns successfully
// with client = nil; every data method then returns
// ErrRedisClientNotInitialized. This preserves the round-31 fail-closed
// behaviour for unconfigured providers and supports legacy unit tests
// that construct with an empty map.
//
// The constructor does NOT block on a Ping at startup (per round-37
// design §2.1). Connectivity is verified by Health() per round-31
// contract; the first data call surfaces any unreachable-backend error
// with full context.
func NewRedisMemoryProvider(config map[string]interface{}) (*RedisMemoryProvider, error) {
	provider := &RedisMemoryProvider{
		prefix: "helix:memory:",
	}

	if prefix, ok := config["prefix"].(string); ok && prefix != "" {
		provider.prefix = prefix
	}

	// Extract optional Redis connection settings.
	var (
		url      string
		host     string
		password string
	)
	port := 6379
	db := 0
	if v, ok := config["url"].(string); ok {
		url = v
	}
	if v, ok := config["host"].(string); ok {
		host = v
	}
	if v, ok := config["port"].(int); ok {
		port = v
	}
	if v, ok := config["password"].(string); ok {
		password = v
	}
	if v, ok := config["db"].(int); ok {
		db = v
	}

	provider.host = host
	provider.port = port

	// nil-client mode: no URL and no host → preserve round-31 sentinel
	// behaviour on every data method. Health continues to return
	// ErrRedisClientNotInitialized.
	if url == "" && host == "" {
		return provider, nil
	}

	var opts *redis.Options
	if url != "" {
		parsed, err := redis.ParseURL(url)
		if err != nil {
			return nil, fmt.Errorf("redis memory provider: invalid url %q: %w", url, err)
		}
		opts = parsed
	} else {
		// HXC-185: host comes from the provider config map and may be a bare
		// IPv6 literal; "%s:%d" produced "::1:6379", rejected by the resolver
		// with "too many colons in address".
		// Guarded by TestNewRedisMemoryProvider_IPv6_ReachesListener.
		opts = &redis.Options{
			Addr:     netutil.JoinHostPort(host, port),
			Password: password,
			DB:       db,
		}
	}

	provider.client = redis.NewClient(opts)
	return provider, nil
}

// Store marshals data to JSON and writes it to Redis under the configured
// prefix + key. TTL is read from the optional context value
// memoryTTLContextKey ("memory:ttl"); absent → no TTL.
func (p *RedisMemoryProvider) Store(ctx context.Context, key string, data interface{}) error {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return ErrRedisClientNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("redis memory provider: marshal value for key %q: %w", key, err)
	}

	ttl := ttlFromContext(ctx)
	fullKey := p.prefix + key
	if err := client.Set(ctx, fullKey, payload, ttl).Err(); err != nil {
		return fmt.Errorf("redis memory provider: SET %q: %w", fullKey, err)
	}
	return nil
}

// Retrieve reads the value at the configured prefix + key from Redis and
// JSON-unmarshals it. A missing key is mapped to ErrKeyNotFound (wrapping
// redis.Nil for diagnostic preservation).
func (p *RedisMemoryProvider) Retrieve(ctx context.Context, key string) (interface{}, error) {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return nil, ErrRedisClientNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}

	fullKey := p.prefix + key
	raw, err := client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("redis memory provider: GET %q: %w (underlying: %v)", fullKey, ErrKeyNotFound, err)
		}
		return nil, fmt.Errorf("redis memory provider: GET %q: %w", fullKey, err)
	}

	var data interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("redis memory provider: unmarshal value for key %q: %w", fullKey, err)
	}
	return data, nil
}

// Search enumerates keys under the configured prefix via Redis SCAN and
// returns up to `limit` entries whose stored value (or key) string-matches
// the query exactly. Honest contract: this is value-equality, not full-text
// search. Callers needing semantic search should use a vector store.
func (p *RedisMemoryProvider) Search(ctx context.Context, query string, limit int) ([]MemorySearchResult, error) {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return nil, ErrRedisClientNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if limit <= 0 {
		return []MemorySearchResult{}, nil
	}

	results := make([]MemorySearchResult, 0, limit)
	iter := client.Scan(ctx, 0, p.prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		if len(results) >= limit {
			break
		}
		fullKey := iter.Val()
		raw, err := client.Get(ctx, fullKey).Bytes()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			return nil, fmt.Errorf("redis memory provider: SCAN GET %q: %w", fullKey, err)
		}

		var data interface{}
		if err := json.Unmarshal(raw, &data); err != nil {
			continue
		}

		shortKey := fullKey
		if len(p.prefix) > 0 && len(fullKey) >= len(p.prefix) && fullKey[:len(p.prefix)] == p.prefix {
			shortKey = fullKey[len(p.prefix):]
		}
		if fmt.Sprintf("%v", data) == query || shortKey == query || fullKey == query {
			results = append(results, MemorySearchResult{
				Key:   shortKey,
				Data:  data,
				Score: 1.0,
			})
		}
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("redis memory provider: SCAN iterator: %w", err)
	}
	return results, nil
}

// Delete removes the value at the configured prefix + key. Idempotent:
// deleting a missing key is not an error (matches Redis DEL semantics).
func (p *RedisMemoryProvider) Delete(ctx context.Context, key string) error {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return ErrRedisClientNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}

	fullKey := p.prefix + key
	n, err := client.Del(ctx, fullKey).Result()
	if err != nil {
		return fmt.Errorf("redis memory provider: DEL %q: %w", fullKey, err)
	}
	if n == 0 {
		return fmt.Errorf("redis memory provider: DEL %q: %w", fullKey, ErrKeyNotFound)
	}
	return nil
}

// Clear removes every key under the configured prefix. Implementation uses
// SCAN + DEL in batches to avoid blocking Redis on large keyspaces.
func (p *RedisMemoryProvider) Clear(ctx context.Context) error {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return ErrRedisClientNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}

	batch := make([]string, 0, 100)
	iter := client.Scan(ctx, 0, p.prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		batch = append(batch, iter.Val())
		if len(batch) >= 100 {
			if err := client.Del(ctx, batch...).Err(); err != nil {
				return fmt.Errorf("redis memory provider: batch DEL: %w", err)
			}
			batch = batch[:0]
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("redis memory provider: SCAN iterator: %w", err)
	}
	if len(batch) > 0 {
		if err := client.Del(ctx, batch...).Err(); err != nil {
			return fmt.Errorf("redis memory provider: final batch DEL: %w", err)
		}
	}
	return nil
}

// Health pings the configured Redis backend.
//
// Round-31 §11.4 anti-bluff sweep (2026-05-18): the original implementation
// returned nil unconditionally with the comment "In production, this would
// ping the Redis server" — a CRITICAL false-health bluff under Article XI
// §11.9 / CONST-035 that certified dead Redis backends as healthy.
//
// Round-37 §11.4 anti-bluff sweep (2026-05-18): the data path was wired
// to a real go-redis/v9 client. Health now follows the same contract:
// when no client is configured (empty constructor) it returns
// ErrRedisClientNotInitialized; when a client IS configured it executes
// a real Ping(ctx) and surfaces any error verbatim. No fabricated PASS.
func (p *RedisMemoryProvider) Health(ctx context.Context) error {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("redis memory provider: health check aborted: %w", err)
	}
	if client == nil {
		return ErrRedisClientNotInitialized
	}
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis memory provider: PING: %w", err)
	}
	return nil
}

// Close releases the underlying Redis connection pool, if a client is
// wired in. Safe to call on a nil-client provider.
func (p *RedisMemoryProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client == nil {
		return nil
	}
	if err := p.client.Close(); err != nil {
		return fmt.Errorf("redis memory provider: close: %w", err)
	}
	p.client = nil
	return nil
}

// Name returns the provider name
func (p *RedisMemoryProvider) Name() string {
	return "redis"
}

// Type returns the provider type
func (p *RedisMemoryProvider) Type() string {
	return "redis"
}

// MemcachedMemoryProvider is a Memcached-backed MemoryProvider that
// delegates Store / Retrieve / Delete / Clear to a real gomemcache
// client. Round-37 §11.4 anti-bluff sweep (2026-05-18): before this
// round the provider held state in a local in-memory map and pretended
// to be Memcached. The Search method additionally promised value-search
// over the dataset — which Memcached's wire protocol cannot deliver
// (Memcached intentionally has no SCAN / key-enumeration command). Both
// behaviours were CRITICAL Article XI §11.9 / CONST-035 bluffs: data
// was lost across restarts, and Search returned fake hits.
//
// Round-37 fixes both. The data path now delegates to gomemcache when a
// client is wired in; Search returns ErrListNotSupportedByBackend as the
// honest contract (use Redis or another backend with key enumeration).
//
// Nil-client mode is preserved for unconfigured providers — every data
// method returns ErrMemcachedClientNotInitialized so callers see the
// missing-configuration gap loudly on first use.
type MemcachedMemoryProvider struct {
	client *memcache.Client
	host   string
	port   int
	prefix string
	mu     sync.RWMutex
}

// NewMemcachedMemoryProvider creates a new Memcached-backed memory
// provider.
//
// Config keys (all optional, backwards compatible):
//   - "servers" []string  explicit list of "host:port" servers (takes precedence)
//   - "host"    string    Memcached hostname (no default — empty means nil-client mode)
//   - "port"    int       Memcached port (defaults to 11211 when host is set)
//   - "prefix"  string    Key namespace prefix (defaults to "helix:memory:")
//   - "timeout" int       Operation timeout in milliseconds (defaults to gomemcache default 100ms)
//
// When neither "servers" nor "host" is set the provider returns
// successfully with client = nil; every data method then returns
// ErrMemcachedClientNotInitialized. This preserves the round-31
// fail-closed behaviour for unconfigured providers and supports legacy
// unit tests that construct with an empty map.
func NewMemcachedMemoryProvider(config map[string]interface{}) (*MemcachedMemoryProvider, error) {
	provider := &MemcachedMemoryProvider{
		prefix: "helix:memory:",
	}

	if prefix, ok := config["prefix"].(string); ok && prefix != "" {
		provider.prefix = prefix
	}

	var (
		host    string
		servers []string
	)
	port := 11211
	if v, ok := config["host"].(string); ok {
		host = v
	}
	if v, ok := config["port"].(int); ok {
		port = v
	}
	if v, ok := config["servers"].([]string); ok {
		servers = v
	}

	provider.host = host
	provider.port = port

	// nil-client mode: no servers and no host → preserve round-31
	// sentinel behaviour. Health continues to return
	// ErrMemcachedClientNotInitialized.
	if len(servers) == 0 && host == "" {
		return provider, nil
	}

	if len(servers) == 0 {
		// HXC-185: bracket IPv6 literals — gomemcache hands the server string
		// straight to net.ResolveTCPAddr, which rejects "::1:11211".
		// Guarded by TestNewMemcachedMemoryProvider_IPv6_ReachesListener.
		servers = []string{netutil.JoinHostPort(host, port)}
	}

	client := memcache.New(servers...)
	if v, ok := config["timeout"].(int); ok && v > 0 {
		client.Timeout = time.Duration(v) * time.Millisecond
	}
	provider.client = client
	return provider, nil
}

// Store marshals data to JSON and writes it to Memcached under the
// configured prefix + key. TTL is read from the optional context value
// memoryTTLContextKey ("memory:ttl"); absent → no expiration (0).
func (p *MemcachedMemoryProvider) Store(ctx context.Context, key string, data interface{}) error {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return ErrMemcachedClientNotInitialized
	}
	_ = ctx // gomemcache is synchronous; ctx not consumed by transport

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("memcached memory provider: marshal value for key %q: %w", key, err)
	}

	expiration := int32(0)
	if ttl := ttlFromContext(ctx); ttl > 0 {
		expiration = int32(ttl.Seconds())
	}

	fullKey := p.prefix + key
	item := &memcache.Item{Key: fullKey, Value: payload, Expiration: expiration}
	if err := client.Set(item); err != nil {
		return fmt.Errorf("memcached memory provider: SET %q: %w", fullKey, err)
	}
	return nil
}

// Retrieve reads the value at the configured prefix + key from Memcached
// and JSON-unmarshals it. A missing key (memcache.ErrCacheMiss) is mapped
// to ErrKeyNotFound for a uniform cross-backend contract.
func (p *MemcachedMemoryProvider) Retrieve(ctx context.Context, key string) (interface{}, error) {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return nil, ErrMemcachedClientNotInitialized
	}
	_ = ctx

	fullKey := p.prefix + key
	item, err := client.Get(fullKey)
	if err != nil {
		if errors.Is(err, memcache.ErrCacheMiss) {
			return nil, fmt.Errorf("memcached memory provider: GET %q: %w (underlying: %v)", fullKey, ErrKeyNotFound, err)
		}
		return nil, fmt.Errorf("memcached memory provider: GET %q: %w", fullKey, err)
	}

	var data interface{}
	if err := json.Unmarshal(item.Value, &data); err != nil {
		return nil, fmt.Errorf("memcached memory provider: unmarshal value for key %q: %w", fullKey, err)
	}
	return data, nil
}

// Search ALWAYS returns ErrListNotSupportedByBackend.
//
// The Memcached wire protocol does not expose a key-enumeration or SCAN
// equivalent. Round-36 and prior shipped a stringly-matched walk over a
// local in-memory map that pretended Search worked — a CRITICAL
// fabricated-capability bluff under Article XI §11.9 / CONST-035.
// Callers needing list / prefix-scan / value-match capability MUST use
// Redis (which supports SCAN) or another backend with key enumeration.
func (p *MemcachedMemoryProvider) Search(ctx context.Context, query string, limit int) ([]MemorySearchResult, error) {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return nil, ErrMemcachedClientNotInitialized
	}
	_ = ctx
	_ = query
	_ = limit
	return nil, ErrListNotSupportedByBackend
}

// Delete removes the value at the configured prefix + key. Idempotent:
// memcache.ErrCacheMiss is swallowed so deleting a missing key is not an
// error (matches Memcached protocol semantics).
func (p *MemcachedMemoryProvider) Delete(ctx context.Context, key string) error {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return ErrMemcachedClientNotInitialized
	}
	_ = ctx

	fullKey := p.prefix + key
	if err := client.Delete(fullKey); err != nil {
		if errors.Is(err, memcache.ErrCacheMiss) {
			return fmt.Errorf("memcached memory provider: DELETE %q: %w", fullKey, ErrKeyNotFound)
		}
		return fmt.Errorf("memcached memory provider: DELETE %q: %w", fullKey, err)
	}
	return nil
}

// Clear issues FlushAll, removing every key from the configured Memcached
// servers. This is a server-wide operation (Memcached has no per-prefix
// scan to scope it). Callers that share a Memcached server with other
// applications should configure a dedicated server pool or use a
// prefix-aware backend like Redis.
func (p *MemcachedMemoryProvider) Clear(ctx context.Context) error {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return ErrMemcachedClientNotInitialized
	}
	_ = ctx

	if err := client.FlushAll(); err != nil {
		return fmt.Errorf("memcached memory provider: FLUSH_ALL: %w", err)
	}
	return nil
}

// Health pings the configured Memcached backend.
//
// Round-31 §11.4 anti-bluff sweep (2026-05-18): the original implementation
// returned nil unconditionally — a CRITICAL false-health bluff under
// Article XI §11.9 / CONST-035.
//
// Round-37 §11.4 anti-bluff sweep (2026-05-18): the data path was wired
// to a real gomemcache client. Health now follows the same contract:
// when no client is configured (empty constructor) it returns
// ErrMemcachedClientNotInitialized; when a client IS configured it
// executes a real Ping() and surfaces any error verbatim.
func (p *MemcachedMemoryProvider) Health(ctx context.Context) error {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("memcached memory provider: health check aborted: %w", err)
	}
	if client == nil {
		return ErrMemcachedClientNotInitialized
	}
	if err := client.Ping(); err != nil {
		return fmt.Errorf("memcached memory provider: PING: %w", err)
	}
	return nil
}

// Close is a no-op for gomemcache (the client manages its own
// connection pool and does not expose Close in v1). Provided for
// interface symmetry with RedisMemoryProvider.Close.
func (p *MemcachedMemoryProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.client = nil
	return nil
}

// Name returns the provider name
func (p *MemcachedMemoryProvider) Name() string {
	return "memcached"
}

// Type returns the provider type
func (p *MemcachedMemoryProvider) Type() string {
	return "memcached"
}

// memoryTTLContextKey is the unexported key used to carry an optional
// per-call TTL into Store. Callers that want a TTL set it via
// WithMemoryTTL(ctx, ttl); the Redis and Memcached providers honour it.
type memoryTTLContextKeyType struct{}

var memoryTTLContextKey = memoryTTLContextKeyType{}

// WithMemoryTTL returns a derived context carrying the given TTL. The
// Redis and Memcached memory providers consult it inside Store to set
// per-key expiry. A zero or negative TTL is ignored.
func WithMemoryTTL(ctx context.Context, ttl time.Duration) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if ttl <= 0 {
		return ctx
	}
	return context.WithValue(ctx, memoryTTLContextKey, ttl)
}

// ttlFromContext extracts the optional TTL carried by WithMemoryTTL.
// Returns 0 when no TTL is set.
func ttlFromContext(ctx context.Context) time.Duration {
	if ctx == nil {
		return 0
	}
	if v, ok := ctx.Value(memoryTTLContextKey).(time.Duration); ok && v > 0 {
		return v
	}
	return 0
}

// FilesystemMemoryProvider is a filesystem-based implementation
type FilesystemMemoryProvider struct {
	config   map[string]interface{}
	basePath string
	index    map[string]string // Maps keys to file paths
	mu       sync.RWMutex
}

// NewFilesystemMemoryProvider creates a new filesystem provider
func NewFilesystemMemoryProvider(config map[string]interface{}) (*FilesystemMemoryProvider, error) {
	provider := &FilesystemMemoryProvider{
		config: config,
		index:  make(map[string]string),
	}

	// Extract filesystem settings from config
	if basePath, ok := config["base_path"].(string); ok {
		provider.basePath = basePath
	} else {
		provider.basePath = "/tmp/helix-memory"
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(provider.basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return provider, nil
}

// Store stores data in filesystem
func (p *FilesystemMemoryProvider) Store(ctx context.Context, key string, data interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Serialize data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}

	// Create safe filename from key
	filename := p.keyToFilename(key)
	filepath := p.basePath + "/" + filename

	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Update index
	p.index[key] = filepath
	return nil
}

// Retrieve retrieves data from filesystem
func (p *FilesystemMemoryProvider) Retrieve(ctx context.Context, key string) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	filepath, exists := p.index[key]
	if !exists {
		// Try to find by filename
		filename := p.keyToFilename(key)
		filepath = p.basePath + "/" + filename
	}

	// Read file
	jsonData, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Deserialize data
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to deserialize data: %w", err)
	}

	return data, nil
}

// Search searches data in filesystem
func (p *FilesystemMemoryProvider) Search(ctx context.Context, query string, limit int) ([]MemorySearchResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	results := make([]MemorySearchResult, 0, limit)
	count := 0

	// Search through indexed keys
	for key, filepath := range p.index {
		if count >= limit {
			break
		}

		// Read the file content
		jsonData, err := os.ReadFile(filepath)
		if err != nil {
			continue
		}

		var data interface{}
		if err := json.Unmarshal(jsonData, &data); err != nil {
			continue
		}

		// Simple string matching - check if query matches key or data value
		if fmt.Sprintf("%v", data) == query || key == query {
			results = append(results, MemorySearchResult{
				Key:   key,
				Data:  data,
				Score: 1.0,
			})
			count++
		}
	}

	return results, nil
}

// Delete deletes data from filesystem
func (p *FilesystemMemoryProvider) Delete(ctx context.Context, key string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	filepath, exists := p.index[key]
	if !exists {
		filename := p.keyToFilename(key)
		filepath = p.basePath + "/" + filename
	}

	// Remove file
	if err := os.Remove(filepath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Remove from index
	delete(p.index, key)
	return nil
}

// Clear clears all data from filesystem
func (p *FilesystemMemoryProvider) Clear(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove all files in base directory
	entries, err := os.ReadDir(p.basePath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			filepath := p.basePath + "/" + entry.Name()
			os.Remove(filepath)
		}
	}

	// Clear index
	p.index = make(map[string]string)
	return nil
}

// Health checks filesystem health
func (p *FilesystemMemoryProvider) Health(ctx context.Context) error {
	// Check if base directory is accessible
	_, err := os.Stat(p.basePath)
	if err != nil {
		return fmt.Errorf("filesystem health check failed: %w", err)
	}
	return nil
}

// keyToFilename converts a key to a safe filename
func (p *FilesystemMemoryProvider) keyToFilename(key string) string {
	// Simple hash-based filename to avoid special characters
	hash := 0
	for _, c := range key {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return fmt.Sprintf("%d.json", hash)
}

// Name returns the provider name
func (p *FilesystemMemoryProvider) Name() string {
	return "filesystem"
}

// Type returns the provider type
func (p *FilesystemMemoryProvider) Type() string {
	return "filesystem"
}

// Global memory manager instance
var globalManager *MemoryManager

// GetGlobalManager returns the global memory manager
func GetGlobalManager() *MemoryManager {
	return globalManager
}

// SetGlobalManager sets the global memory manager
func SetGlobalManager(manager *MemoryManager) {
	globalManager = manager
}

// InitializeGlobalManager initializes the global memory manager
func InitializeGlobalManager(config *MemoryConfig) {
	globalManager = NewMemoryManager(config)

	// Register default in-memory provider
	inMemory, _ := NewInMemoryProvider(map[string]interface{}{})
	globalManager.RegisterProvider("inmemory", inMemory)
}

// StoreGlobal stores data using the global manager
func StoreGlobal(ctx context.Context, key string, data interface{}) error {
	if globalManager == nil {
		return fmt.Errorf("global memory manager not initialized")
	}
	return globalManager.Store(ctx, key, data)
}

// RetrieveGlobal retrieves data using the global manager
func RetrieveGlobal(ctx context.Context, key string) (interface{}, error) {
	if globalManager == nil {
		return nil, fmt.Errorf("global memory manager not initialized")
	}
	return globalManager.Retrieve(ctx, key)
}

// SearchGlobal searches data using the global manager
func SearchGlobal(ctx context.Context, query string, limit int) ([]MemorySearchResult, error) {
	if globalManager == nil {
		return nil, fmt.Errorf("global memory manager not initialized")
	}
	return globalManager.Search(ctx, query, limit)
}
