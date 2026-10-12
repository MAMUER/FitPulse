package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	require.NoError(t, client.Set(ctx, "key1", "val1", 10*time.Second))
	require.NoError(t, client.Set(ctx, "key2", "val2", 10*time.Second))

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

func TestNewClientFromValkey(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = rdb.Close() }()

	client := NewClientFromValkey(rdb)
	require.NotNil(t, client)
	assert.NotNil(t, client.rdb)

	ctx := context.Background()
	require.NoError(t, client.Set(ctx, "valkey_key", "value", time.Minute))

	val, err := client.Get(ctx, "valkey_key")
	require.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestNewClientFromValkeyDelegatesToFromRedisClient(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = rdb.Close() }()

	assert.Same(t, FromRedisClient(rdb).rdb, NewClientFromValkey(rdb).rdb)
}

func TestDelWithoutKeys(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	assert.NoError(t, client.Del(context.Background()))
}

func TestCloseWithNilRedisClient(t *testing.T) {
	client := &Client{}

	assert.NoError(t, client.Close())
}

func TestCloseReturnsErrorFromClosedRedisClient(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	client := FromRedisClient(rdb)

	require.NoError(t, rdb.Close())

	err = client.Close()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close valkey client")
}

func setupFastFailValkey(t *testing.T) (*Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := &Client{
		rdb: redis.NewClient(&redis.Options{
			Addr:         mr.Addr(),
			MaxRetries:   -1,
			DialTimeout:  200 * time.Millisecond,
			ReadTimeout:  500 * time.Millisecond,
			WriteTimeout: 500 * time.Millisecond,
		}),
	}
	return client, mr
}

func TestCacheOperationsAfterBackingStoreClosed(t *testing.T) {
	client, mr := setupFastFailValkey(t)
	require.NoError(t, client.Set(context.Background(), "closed_key", "value", time.Minute))
	mr.Close()

	ctx := context.Background()

	t.Run("set", func(t *testing.T) {
		err := client.Set(ctx, "closed_key", "value", time.Minute)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache set:")
	})

	t.Run("get", func(t *testing.T) {
		val, err := client.Get(ctx, "closed_key")
		require.Error(t, err)
		assert.Empty(t, val)
		assert.Contains(t, err.Error(), "cache get:")
		assert.NotErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("get decoded", func(t *testing.T) {
		var out map[string]string
		err := client.GetDecoded(ctx, "closed_key", &out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache get decoded:")
		assert.NotErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("set decoded", func(t *testing.T) {
		err := client.SetDecoded(ctx, "closed_key", map[string]string{"a": "b"}, time.Minute)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache set decoded:")
	})

	t.Run("get or set decoded", func(t *testing.T) {
		var out map[string]string
		err := client.GetOrSetDecoded(ctx, "closed_key", time.Minute, func() (interface{}, error) {
			return map[string]string{"a": "b"}, nil
		}, &out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache get decoded:")
	})

	t.Run("del", func(t *testing.T) {
		err := client.Del(ctx, "closed_key")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache del:")
	})

	t.Run("exists", func(t *testing.T) {
		exists, err := client.Exists(ctx, "closed_key")
		require.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "cache exists:")
	})

	t.Run("ttl", func(t *testing.T) {
		ttl, err := client.TTL(ctx, "closed_key")
		require.Error(t, err)
		assert.Equal(t, time.Duration(0), ttl)
		assert.Contains(t, err.Error(), "cache ttl:")
	})

	t.Run("refresh ttl", func(t *testing.T) {
		err := client.RefreshTTL(ctx, "closed_key", time.Minute)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache refresh ttl:")
	})
}

func TestSetDecodedMarshalError(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	err := client.SetDecoded(context.Background(), "unencodable", make(chan int), time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal value:")

	err = client.GetOrSetDecoded(context.Background(), "unencodable", time.Minute, func() (interface{}, error) {
		return make(chan int), nil
	}, new(interface{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal value:")
}

func TestGetOrSetDecodedKeyValidation(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	var out string
	err := client.GetOrSetDecoded(context.Background(), "", time.Minute, func() (interface{}, error) {
		t.Fatal("fetch must not be called for an invalid key")
		return nil, nil
	}, &out)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestGetOrSetDecodedFetchError(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	expected := errors.New("fetch failed")

	var out string
	err := client.GetOrSetDecoded(context.Background(), "fetch_error", time.Minute, func() (interface{}, error) {
		return nil, expected
	}, &out)

	require.Error(t, err)
	assert.ErrorIs(t, err, expected)
	assert.Empty(t, out)
}

func TestGetOrSetDecodedNonNotFoundErrorFromCache(t *testing.T) {
	client, mr := setupFastFailValkey(t)
	require.NoError(t, client.Set(context.Background(), "present", "value", time.Minute))
	mr.Close()

	fetchCalls := 0
	var out string
	err := client.GetOrSetDecoded(context.Background(), "present", time.Minute, func() (interface{}, error) {
		fetchCalls++
		return "fetched", nil
	}, &out)

	require.Error(t, err)
	assert.Equal(t, 0, fetchCalls)
	assert.Empty(t, out)
}

func TestGetDecodedInvalidJSON(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()
	require.NoError(t, client.Set(ctx, "raw_key", "not-json", time.Minute))

	var out map[string]string
	err := client.GetDecoded(ctx, "raw_key", &out)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrKeyNotFound)
}

func TestTTLOfKeyWithoutExpiration(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()
	require.NoError(t, client.Set(ctx, "no_expiry_key", "value", 0))

	ttl, err := client.TTL(ctx, "no_expiry_key")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), ttl)

	ttl, err = client.TTL(ctx, "missing_key")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-2), ttl)
}

func TestDelIgnoresMissingKeys(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	assert.NoError(t, client.Del(context.Background(), "missing1", "missing2"))
}

func TestSetDecodedOverwritesExistingValue(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()
	require.NoError(t, client.SetDecoded(ctx, "counter", map[string]int{"count": 1}, time.Minute))
	require.NoError(t, client.SetDecoded(ctx, "counter", map[string]int{"count": 2}, time.Minute))

	out := map[string]int{}
	require.NoError(t, client.GetDecoded(ctx, "counter", &out))
	assert.Equal(t, map[string]int{"count": 2}, out)
}

func TestCacheConcurrentAccess(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()
	type payload struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	var wg sync.WaitGroup
	const workers = 8
	const iterations = 25

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("concurrent_%d", worker)

				err := client.Set(ctx, key, worker, time.Minute)
				assert.NoError(t, err)

				val, err := client.Get(ctx, key)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("%d", worker), val)

				stored := payload{Name: key, Value: worker}
				err = client.SetDecoded(ctx, key+"_json", stored, time.Minute)
				assert.NoError(t, err)

				var decoded payload
				err = client.GetDecoded(ctx, key+"_json", &decoded)
				assert.NoError(t, err)
				assert.Equal(t, stored, decoded)

				exists, err := client.Exists(ctx, key)
				assert.NoError(t, err)
				assert.True(t, exists)

				ttl, err := client.TTL(ctx, key)
				assert.NoError(t, err)
				assert.Greater(t, ttl, time.Duration(0))

				assert.NoError(t, client.RefreshTTL(ctx, key, time.Minute))

				var fetched payload
				fetchCalls := 0
				err = client.GetOrSetDecoded(ctx, key+"_fetch", time.Minute, func() (interface{}, error) {
					fetchCalls++
					return stored, nil
				}, &fetched)
				assert.NoError(t, err)
				assert.Equal(t, stored, fetched)

				assert.NoError(t, client.Del(ctx, key))
			}
		}(w)
	}

	wg.Wait()
}

func TestConcurrentGetOrSetDecodedFetchesOnce(t *testing.T) {
	client, mr := setupTestValkey(t)
	defer mr.Close()

	ctx := context.Background()
	const workers = 8

	var mu sync.Mutex
	fetchCalls := 0
	fetchFn := func() (interface{}, error) {
		mu.Lock()
		fetchCalls++
		mu.Unlock()
		time.Sleep(5 * time.Millisecond)
		return map[string]int{"value": 1}, nil
	}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out map[string]int
			err := client.GetOrSetDecoded(ctx, "shared_fetch", time.Minute, fetchFn, &out)
			assert.NoError(t, err)
			assert.Equal(t, map[string]int{"value": 1}, out)
		}()
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, workers, fetchCalls)

	var out map[string]int
	require.NoError(t, client.GetOrSetDecoded(ctx, "shared_fetch", time.Minute, fetchFn, &out))
	assert.Equal(t, workers, fetchCalls)
}
