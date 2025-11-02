package redis

import (
	"context"
	"fmt"
	"time"

	"dev.helix.code/internal/config"
	"github.com/go-redis/redis/v8"
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

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
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

// IsEnabled returns whether Redis is enabled
func (c *Client) IsEnabled() bool {
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
		return "", fmt.Errorf("Redis is disabled")
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
		return 0, fmt.Errorf("Redis is disabled")
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
		return 0, fmt.Errorf("Redis is disabled")
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
		return "", fmt.Errorf("Redis is disabled")
	}
	return c.client.HGet(ctx, key, field).Result()
}

// HGetAll gets all the fields and values in a hash
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("Redis is disabled")
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
		return "", fmt.Errorf("Redis is disabled")
	}
	return c.client.RPop(ctx, key).Result()
}

// BRPop is a blocking list pop primitive
func (c *Client) BRPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("Redis is disabled")
	}
	return c.client.BRPop(ctx, timeout, keys...).Result()
}

// LLen returns the length of a list
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	if !c.IsEnabled() {
		return 0, fmt.Errorf("Redis is disabled")
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
		return nil, fmt.Errorf("Redis is disabled")
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
func (c *Client) ZAdd(ctx context.Context, key string, members ...*redis.Z) error {
	if !c.IsEnabled() {
		return nil // No-op if Redis is disabled
	}
	return c.client.ZAdd(ctx, key, members...).Err()
}

// ZRange returns members of a sorted set
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("Redis is disabled")
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
