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

func TestNewClientWithConfig(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	ctx := context.Background()
	client, err := NewClientWithConfig(ctx, Config{
		Addr:         mr.Addr(),
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	})
	assert.NoError(t, err)
	assert.NotNil(t, client)

	err = client.Set(ctx, "config_test", "value", 10*time.Second)
	assert.NoError(t, err)
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
	assert.ErrorIs(t, err, ErrKeyNotFound)
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

	mr.FastForward(1100 * time.Millisecond)

	_, err = client.Get(ctx, "expire_key")
	assert.Error(t, err, "Key should have expired")
}

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
		{"bool_true", "key4", true, "1"},
		{"bool_false", "key5", false, "0"},
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

	defer func() { _ = client.Close() }()

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

func TestKeyValidation(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()

	_, err := client.Get(ctx, "")
	assert.ErrorIs(t, err, ErrInvalidKey)

	err = client.Set(ctx, "", "value", 10*time.Second)
	assert.ErrorIs(t, err, ErrInvalidKey)

	err = client.Del(ctx, "")
	assert.ErrorIs(t, err, ErrInvalidKey)

	var v interface{}
	err = client.GetDecoded(ctx, "", &v)
	assert.ErrorIs(t, err, ErrInvalidKey)

	err = client.SetDecoded(ctx, "", map[string]string{"a": "b"}, 10*time.Second)
	assert.ErrorIs(t, err, ErrInvalidKey)

	_, err = client.Exists(ctx, "")
	assert.ErrorIs(t, err, ErrInvalidKey)

	_, err = client.TTL(ctx, "")
	assert.ErrorIs(t, err, ErrInvalidKey)

	err = client.RefreshTTL(ctx, "", 10*time.Second)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestGetDecodedSetDecoded(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := TestStruct{Name: "test", Value: 42}
	err := client.SetDecoded(ctx, "struct_key", original, 10*time.Second)
	assert.NoError(t, err)

	var decoded TestStruct
	err = client.GetDecoded(ctx, "struct_key", &decoded)
	assert.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestGetDecodedNotFound(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()

	var decoded interface{}
	err := client.GetDecoded(ctx, "nonexistent", &decoded)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestGetOrSetDecoded(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	fetchCount := 0
	fetchFn := func() (interface{}, error) {
		fetchCount++
		return TestStruct{Name: "fetched", Value: 100}, nil
	}

	var val TestStruct
	err := client.GetOrSetDecoded(ctx, "or_set_key", 10*time.Second, fetchFn, &val)
	assert.NoError(t, err)
	assert.Equal(t, 1, fetchCount)
	assert.Equal(t, "fetched", val.Name)
	assert.Equal(t, 100, val.Value)

	err = client.GetOrSetDecoded(ctx, "or_set_key", 10*time.Second, fetchFn, &val)
	assert.NoError(t, err)
	assert.Equal(t, 1, fetchCount)
}

func TestExists(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()

	exists, err := client.Exists(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)

	err = client.Set(ctx, "exists_key", "value", 10*time.Second)
	assert.NoError(t, err)

	exists, err = client.Exists(ctx, "exists_key")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestTTL(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()

	err := client.Set(ctx, "ttl_key", "value", 10*time.Second)
	assert.NoError(t, err)

	ttl, err := client.TTL(ctx, "ttl_key")
	assert.NoError(t, err)
	assert.Greater(t, ttl, 9*time.Second)
	assert.LessOrEqual(t, ttl, 10*time.Second)

	_, err = client.TTL(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestRefreshTTL(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()

	err := client.Set(ctx, "refresh_key", "value", 1*time.Second)
	assert.NoError(t, err)

	ttl, err := client.TTL(ctx, "refresh_key")
	assert.NoError(t, err)
	assert.Less(t, ttl, 2*time.Second)

	err = client.RefreshTTL(ctx, "refresh_key", 10*time.Second)
	assert.NoError(t, err)

	ttl, err = client.TTL(ctx, "refresh_key")
	assert.NoError(t, err)
	assert.Greater(t, ttl, 9*time.Second)
	assert.LessOrEqual(t, ttl, 10*time.Second)
}

func TestFromRedisClient(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = rdb.Close() }()

	client := FromRedisClient(rdb)
	assert.NotNil(t, client)
	assert.NotNil(t, client.rdb)
}
