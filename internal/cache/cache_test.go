// internal/cache/cache_test.go
package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestValkey(t *testing.T) (*Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := &Client{
		rdb: redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		}),
	}
	return client, mr
}

func TestNewClient(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client, err := NewClient(mr.Addr(), "", 0)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.rdb)
}

func TestNewClientInvalidAddr(t *testing.T) {
	client, err := NewClient("invalid:6379", "", 0)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestSetAndGet(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()
	err := client.Set(ctx, "test_key", "test_value", 10*time.Second)
	assert.NoError(t, err)

	val, err := client.Get(ctx, "test_key")
	assert.NoError(t, err)
	assert.Equal(t, "test_value", val)
}

func TestGetNonExistent(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()
	val, err := client.Get(ctx, "non_existent")
	assert.Error(t, err)
	assert.Empty(t, val)
}

func TestDel(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()
	_ = client.Set(ctx, "key1", "val1", 10*time.Second)
	_ = client.Set(ctx, "key2", "val2", 10*time.Second)

	err := client.Del(ctx, "key1", "key2")
	assert.NoError(t, err)

	_, err = client.Get(ctx, "key1")
	assert.Error(t, err)
	_, err = client.Get(ctx, "key2")
	assert.Error(t, err)
}

func TestSetWithExpiration(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()
	err := client.Set(ctx, "expire_key", "value", 1*time.Second)
	assert.NoError(t, err)

	val, err := client.Get(ctx, "expire_key")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)

	// In miniredis, manually advance time
	mr.FastForward(1100 * time.Millisecond)

	_, err = client.Get(ctx, "expire_key")
	assert.Error(t, err, "Key should have expired")
}

// Valkey stores bool as "1"/"0"
func TestSetMultipleTypes(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client, err := NewClient(mr.Addr(), "", 0)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	tests := []struct {
		name     string
		key      string
		value    interface{}
		expected string
	}{
		{"string", "key1", "val1", "val1"},
		{"int", "key2", 42, "42"},
		{"float", "key3", 3.14, "3.14"},
		{"bool_true", "key4", true, "1"},   // Valkey: true -> "1"
		{"bool_false", "key5", false, "0"}, // Valkey: false -> "0"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.Set(ctx, tt.key, tt.value, 10*time.Second)
			require.NoError(t, err)

			val, err := client.Get(ctx, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func TestClientClose(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client, err := NewClient(mr.Addr(), "", 0)
	require.NoError(t, err)

	// Close() doesn't return a value, just call it
	defer func() { _ = client.Close() }()

	// Repeated Close should not panic
	defer func() { _ = client.Close() }()
}

func TestSetWithNilValue(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()
	err := client.Set(ctx, "nil_key", nil, 10*time.Second)
	assert.NoError(t, err)

	val, err := client.Get(ctx, "nil_key")
	assert.NoError(t, err)
	assert.Equal(t, "", val)
}
