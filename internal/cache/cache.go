// Package cache provides a type-safe, observable caching abstraction over Valkey.
// It uses go-redis/v9, which is wire-compatible with Valkey, and supports JSON
// serialization, key validation, Prometheus metrics, and common cache patterns.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
)

var (
	// ErrKeyNotFound is returned when a cache key does not exist.
	ErrKeyNotFound = errors.New("cache key not found")
	// ErrInvalidKey is returned when a cache key is empty.
	ErrInvalidKey = errors.New("cache key cannot be empty")
)

var (
	cacheRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_requests_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation"},
	)
	cacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
	)
	cacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
	)
	cacheErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_errors_total",
			Help: "Total number of cache errors",
		},
		[]string{"operation"},
	)
	cacheLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_latency_seconds",
			Help:    "Cache operation latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

// Config holds configuration for the cache client.
type Config struct {
	// Addr is the Valkey server address.
	Addr string
	// Password is the Valkey password.
	Password string
	// DB is the Valkey database number.
	DB int
	// PoolSize is the maximum number of socket connections.
	PoolSize int
	// MinIdleConns is the minimum number of idle connections.
	MinIdleConns int
	// DialTimeout is the timeout for establishing new connections.
	DialTimeout time.Duration
	// ReadTimeout is the timeout for socket reads.
	ReadTimeout time.Duration
	// WriteTimeout is the timeout for socket writes.
	WriteTimeout time.Duration
	// PoolTimeout is the timeout for waiting for a connection from the pool.
	PoolTimeout time.Duration
}

// Client is a Valkey cache client.
// It uses go-redis/v9, which is wire-compatible with Valkey.
type Client struct {
	rdb *redis.Client
}

// NewClientWithConfig creates a new cache client with the given configuration.
func NewClientWithConfig(ctx context.Context, cfg Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolTimeout:  cfg.PoolTimeout,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping valkey: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

// NewClient creates a new cache client.
func NewClient(addr, password string, db int) (*Client, error) {
	return NewClientWithConfig(context.Background(), Config{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

// FromRedisClient creates a Cache Client from an existing Redis/Valkey client.
func FromRedisClient(rdb *redis.Client) *Client {
	return &Client{rdb: rdb}
}

// NewClientFromValkey creates a Cache Client from an existing Valkey client.
// Valkey is wire-compatible with Redis, so this accepts a *redis.Client.
func NewClientFromValkey(rdb *redis.Client) *Client {
	return FromRedisClient(rdb)
}

func (c *Client) validateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	return nil
}

func observeLatency(operation string, start time.Time) {
	cacheLatency.WithLabelValues(operation).Observe(time.Since(start).Seconds())
}

// Set stores a value in the cache with the given expiration.
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := c.validateKey(key); err != nil {
		return err
	}
	start := time.Now()
	defer observeLatency("set", start)
	if err := c.rdb.Set(ctx, key, value, expiration).Err(); err != nil {
		cacheRequests.WithLabelValues("set").Inc()
		cacheErrors.WithLabelValues("set").Inc()
		return fmt.Errorf("cache set: %w", err)
	}
	cacheRequests.WithLabelValues("set").Inc()
	return nil
}

// Get retrieves a value from the cache.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if err := c.validateKey(key); err != nil {
		return "", err
	}
	start := time.Now()
	defer observeLatency("get", start)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			cacheRequests.WithLabelValues("get").Inc()
			cacheMisses.Inc()
			return "", ErrKeyNotFound
		}
		cacheRequests.WithLabelValues("get").Inc()
		cacheErrors.WithLabelValues("get").Inc()
		return val, fmt.Errorf("cache get: %w", err)
	}
	cacheRequests.WithLabelValues("get").Inc()
	cacheHits.Inc()
	return val, nil
}

// GetDecoded retrieves a JSON-encoded value from the cache and decodes it into v.
func (c *Client) GetDecoded(ctx context.Context, key string, v interface{}) error {
	if err := c.validateKey(key); err != nil {
		return err
	}
	start := time.Now()
	defer observeLatency("get_decoded", start)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			cacheRequests.WithLabelValues("get_decoded").Inc()
			cacheMisses.Inc()
			return ErrKeyNotFound
		}
		cacheRequests.WithLabelValues("get_decoded").Inc()
		cacheErrors.WithLabelValues("get_decoded").Inc()
		return fmt.Errorf("cache get decoded: %w", err)
	}
	cacheRequests.WithLabelValues("get_decoded").Inc()
	cacheHits.Inc()
	return json.Unmarshal([]byte(val), v)
}

// SetDecoded JSON-encodes value and stores it in the cache.
func (c *Client) SetDecoded(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := c.validateKey(key); err != nil {
		return err
	}
	start := time.Now()
	defer observeLatency("set_decoded", start)
	data, err := json.Marshal(value)
	if err != nil {
		cacheRequests.WithLabelValues("set_decoded").Inc()
		cacheErrors.WithLabelValues("set_decoded").Inc()
		return fmt.Errorf("marshal value: %w", err)
	}
	if err := c.rdb.Set(ctx, key, data, expiration).Err(); err != nil {
		cacheRequests.WithLabelValues("set_decoded").Inc()
		cacheErrors.WithLabelValues("set_decoded").Inc()
		return fmt.Errorf("cache set decoded: %w", err)
	}
	cacheRequests.WithLabelValues("set_decoded").Inc()
	return nil
}

// GetOrSetDecoded implements a cache-aside pattern.
// It tries to get the value from cache; on miss, it calls fetchFn, stores the result, and decodes it into v.
func (c *Client) GetOrSetDecoded(ctx context.Context, key string, expiration time.Duration, fetchFn func() (interface{}, error), v interface{}) error {
	if err := c.validateKey(key); err != nil {
		return err
	}

	err := c.GetDecoded(ctx, key, v)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrKeyNotFound) {
		return err
	}

	value, err := fetchFn()
	if err != nil {
		return err
	}

	if err := c.SetDecoded(ctx, key, value, expiration); err != nil {
		return err
	}

	return json.Unmarshal(mustMarshal(value), v)
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// Del deletes one or more keys from the cache.
func (c *Client) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	for _, key := range keys {
		if err := c.validateKey(key); err != nil {
			return err
		}
	}
	start := time.Now()
	defer observeLatency("del", start)
	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		cacheRequests.WithLabelValues("del").Inc()
		cacheErrors.WithLabelValues("del").Inc()
		return fmt.Errorf("cache del: %w", err)
	}
	cacheRequests.WithLabelValues("del").Inc()
	return nil
}

// Exists checks whether a key exists in the cache.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	if err := c.validateKey(key); err != nil {
		return false, err
	}
	start := time.Now()
	defer observeLatency("exists", start)
	n, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		cacheRequests.WithLabelValues("exists").Inc()
		cacheErrors.WithLabelValues("exists").Inc()
		return false, fmt.Errorf("cache exists: %w", err)
	}
	cacheRequests.WithLabelValues("exists").Inc()
	return n > 0, nil
}

// TTL returns the remaining TTL for a key.
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	if err := c.validateKey(key); err != nil {
		return 0, err
	}
	start := time.Now()
	defer observeLatency("ttl", start)
	duration, err := c.rdb.TTL(ctx, key).Result()
	if err != nil {
		cacheRequests.WithLabelValues("ttl").Inc()
		cacheErrors.WithLabelValues("ttl").Inc()
		return 0, fmt.Errorf("cache ttl: %w", err)
	}
	cacheRequests.WithLabelValues("ttl").Inc()
	return duration, nil
}

// RefreshTTL extends the TTL for an existing key.
func (c *Client) RefreshTTL(ctx context.Context, key string, expiration time.Duration) error {
	if err := c.validateKey(key); err != nil {
		return err
	}
	start := time.Now()
	defer observeLatency("refresh_ttl", start)
	if err := c.rdb.Expire(ctx, key, expiration).Err(); err != nil {
		cacheRequests.WithLabelValues("refresh_ttl").Inc()
		cacheErrors.WithLabelValues("refresh_ttl").Inc()
		return fmt.Errorf("cache refresh ttl: %w", err)
	}
	cacheRequests.WithLabelValues("refresh_ttl").Inc()
	return nil
}

// Close closes the cache client.
func (c *Client) Close() error {
	if c.rdb != nil {
		if err := c.rdb.Close(); err != nil {
			return fmt.Errorf("close valkey client: %w", err)
		}
	}
	return nil
}
