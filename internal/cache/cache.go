package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

// Close closes the cache client and returns any error encountered.
func (c *Client) Close() error {
	if c.rdb != nil {
		if err := c.rdb.Close(); err != nil {
			return fmt.Errorf("close valkey client: %w", err)
		}
	}
	return nil
}

func NewClient(addr, password string, db int) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping valkey: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

// NewClientFromValkey creates a Cache Client from an existing Valkey client
func NewClientFromValkey(rdb *redis.Client) *Client {
	return &Client{rdb: rdb}
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := c.rdb.Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	return nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return value, fmt.Errorf("cache get: %w", err)
	}
	return value, nil
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("cache del: %w", err)
	}
	return nil
}
