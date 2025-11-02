package redis

import (
	"context"
	"testing"
	"time"

	"dev.helix.code/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewClient_Disabled(t *testing.T) {
	cfg := &config.RedisConfig{
		Enabled: false,
		Host:    "localhost",
		Port:    6379,
	}

	client, err := NewClient(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.False(t, client.IsEnabled())
	assert.Nil(t, client.GetClient())
}

func TestNewClient_InvalidConfig(t *testing.T) {
	cfg := &config.RedisConfig{
		Enabled: true,
		Host:    "invalid-host",
		Port:    6379,
	}

	client, err := NewClient(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

func TestClient_Methods_Disabled(t *testing.T) {
	cfg := &config.RedisConfig{Enabled: false}
	client, _ := NewClient(cfg)
	ctx := context.Background()

	// Test methods that should be no-ops when disabled
	assert.NoError(t, client.Set(ctx, "key", "value", 0))
	assert.NoError(t, client.Del(ctx, "key"))
	assert.NoError(t, client.Expire(ctx, "key", time.Hour))
	assert.NoError(t, client.HSet(ctx, "key", "field", "value"))
	assert.NoError(t, client.HDel(ctx, "key", "field"))
	assert.NoError(t, client.Publish(ctx, "channel", "message"))
	assert.NoError(t, client.LPush(ctx, "key", "value"))
	assert.NoError(t, client.SAdd(ctx, "key", "member"))
	assert.NoError(t, client.SRem(ctx, "key", "member"))
	assert.NoError(t, client.ZAdd(ctx, "key"))
	assert.NoError(t, client.ZRem(ctx, "key", "member"))

	// Test methods that should return errors when disabled
	_, err := client.Get(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Redis is disabled")

	_, err = client.Exists(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Redis is disabled")

	_, err = client.TTL(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Redis is disabled")

	_, err = client.HGet(ctx, "key", "field")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Redis is disabled")

	_, err = client.HGetAll(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Redis is disabled")

	_, err = client.RPop(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Redis is disabled")

	_, err = client.BRPop(ctx, time.Second, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Redis is disabled")

	_, err = client.LLen(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Redis is disabled")

	_, err = client.SMembers(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Redis is disabled")

	_, err = client.ZRange(ctx, "key", 0, -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Redis is disabled")

	// Test Subscribe (should return nil when disabled)
	pubsub := client.Subscribe(ctx, "channel")
	assert.Nil(t, pubsub)
}

func TestClient_Close(t *testing.T) {
	// Test Close on disabled client
	cfg := &config.RedisConfig{Enabled: false}
	client, _ := NewClient(cfg)
	assert.NoError(t, client.Close())

	// Test Close on client with nil client
	client = &Client{client: nil, config: &config.RedisConfig{Enabled: true}}
	assert.NoError(t, client.Close())
}
