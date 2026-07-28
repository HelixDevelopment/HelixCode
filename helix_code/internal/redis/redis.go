package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dev.helix.code/internal/config"
	"dev.helix.code/internal/netutil"
	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client
type Client struct {
	client *redis.Client
	config *config.RedisConfig
}

// NewClient creates a new Redis client
func NewClient(cfg *config.RedisConfig) (*Client, error) {
	if !cfg.Enabled {
		return &Client{config: cfg}, nil
	}

	// CONST-035: validate non-empty host before attempting connection.
	// Otherwise `Sprintf("%s:%d", "", 6379)` produces ":6379" which
	// resolves to localhost:6379 if a Redis is running — making the
	// caller's empty-host bug silently "succeed". The
	// TestNewClient_EmptyHost test relies on this validation to FAIL
	// fast on misconfigured callers; without it, the test only passes
	// when no local Redis is running (host-dependent flake).
	if cfg.Host == "" {
		return nil, errors.New(tr(context.Background(), "internal_redis_empty_host", nil))
	}

	// HXC-185: cfg.Host is operator-supplied, so it can be a bare IPv6
	// literal. "%s:%d" produced "::1:6379", which the resolver rejects with
	// "too many colons in address" — the client never dialled at all.
	// Guarded by TestNewClient_IPv6Host_ReachesRealListener.
	rdb := redis.NewClient(&redis.Options{
		Addr:     netutil.JoinHostPort(cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.Database,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", tr(ctx, "internal_redis_failed_connect", map[string]any{"Err": err.Error()}), err)
	}

	return &Client{
		client: rdb,
		config: cfg,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// IsEnabled returns whether Redis is enabled.
//
// Nil-receiver-safe: a nil *Client (e.g. server.New(cfg, db, nil)) reports
// disabled rather than panicking on the c.config dereference. Every other
// method on Client already funnels through IsEnabled() before touching
// c.client, so guarding here makes the whole type robust to a nil receiver.
func (c *Client) IsEnabled() bool {
	if c == nil {
		return false
	}
	return c.config != nil && c.config.Enabled
}

// GetClient returns the underlying Redis client
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// Set sets a key-value pair
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get gets a value by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if !c.IsEnabled() {
		return "", errors.New(tr(ctx, "internal_redis_disabled", nil))
	}
	return c.client.Get(ctx, key).Result()
}

// Del deletes keys
func (c *Client) Del(ctx context.Context, keys ...string) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.Del(ctx, keys...).Err()
}

// Exists checks if keys exist
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	if !c.IsEnabled() {
		return 0, errors.New(tr(ctx, "internal_redis_disabled", nil))
	}
	return c.client.Exists(ctx, keys...).Result()
}

// Expire sets expiration on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.Expire(ctx, key, expiration).Err()
}

// TTL gets the time to live for a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	if !c.IsEnabled() {
		return 0, errors.New(tr(ctx, "internal_redis_disabled", nil))
	}
	return c.client.TTL(ctx, key).Result()
}

// HSet sets field in the hash stored at key
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.HSet(ctx, key, values...).Err()
}

// HGet gets the value of a hash field
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	if !c.IsEnabled() {
		return "", errors.New(tr(ctx, "internal_redis_disabled", nil))
	}
	return c.client.HGet(ctx, key, field).Result()
}

// HGetAll gets all the fields and values in a hash
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if !c.IsEnabled() {
		return nil, errors.New(tr(ctx, "internal_redis_disabled", nil))
	}
	return c.client.HGetAll(ctx, key).Result()
}

// HDel deletes fields from the hash stored at key
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.HDel(ctx, key, fields...).Err()
}

// Publish publishes a message to a channel
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to channels
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	if !c.IsEnabled() {
		return nil
	}
	return c.client.Subscribe(ctx, channels...)
}

// LPush prepends values to a list
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.LPush(ctx, key, values...).Err()
}

// RPop removes and returns the last element of the list
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	if !c.IsEnabled() {
		return "", errors.New(tr(ctx, "internal_redis_disabled", nil))
	}
	return c.client.RPop(ctx, key).Result()
}

// BRPop is a blocking list pop primitive
func (c *Client) BRPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	if !c.IsEnabled() {
		return nil, errors.New(tr(ctx, "internal_redis_disabled", nil))
	}
	return c.client.BRPop(ctx, timeout, keys...).Result()
}

// LLen returns the length of a list
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	if !c.IsEnabled() {
		return 0, errors.New(tr(ctx, "internal_redis_disabled", nil))
	}
	return c.client.LLen(ctx, key).Result()
}

// SAdd adds members to a set
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.SAdd(ctx, key, members...).Err()
}

// SMembers returns all members of a set
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	if !c.IsEnabled() {
		return nil, errors.New(tr(ctx, "internal_redis_disabled", nil))
	}
	return c.client.SMembers(ctx, key).Result()
}

// SRem removes members from a set
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.SRem(ctx, key, members...).Err()
}

// ZAdd adds members to a sorted set
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.ZAdd(ctx, key, members...).Err()
}

// ZRange returns members of a sorted set
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	if !c.IsEnabled() {
		return nil, errors.New(tr(ctx, "internal_redis_disabled", nil))
	}
	return c.client.ZRange(ctx, key, start, stop).Result()
}

// ZRem removes members from a sorted set
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.ZRem(ctx, key, members...).Err()
}
