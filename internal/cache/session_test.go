package cache

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSessionTest(t *testing.T) (*SessionStore, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := &Client{
		rdb: redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		}),
	}
	store := NewSessionStore(client)
	return store, mr
}

// --- AuthCode tests ---

func TestCreateAuthCode(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	code, err := store.CreateAuthCode(context.Background(), "user-1", "client-1", "http://localhost/callback")
	require.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Len(t, code, 43) // base64url encoded 32 bytes truncated
}

func TestExchangeAuthCodeSuccess(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	code, err := store.CreateAuthCode(context.Background(), "user-1", "client-1", "http://localhost/callback")
	require.NoError(t, err)

	userID, err := store.ExchangeAuthCode(context.Background(), code, "client-1", "http://localhost/callback")
	require.NoError(t, err)
	assert.Equal(t, "user-1", userID)
}

func TestExchangeAuthCodeNotFound(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	_, err := store.ExchangeAuthCode(context.Background(), "nonexistent", "client-1", "http://localhost/callback")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExchangeAuthCodeClientIDMismatch(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	code, err := store.CreateAuthCode(context.Background(), "user-1", "client-1", "http://localhost/callback")
	require.NoError(t, err)

	_, err = store.ExchangeAuthCode(context.Background(), code, "client-2", "http://localhost/callback")
	assert.Error(t, err)
}

func TestExchangeAuthCodeRedirectMismatch(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	code, err := store.CreateAuthCode(context.Background(), "user-1", "client-1", "http://localhost/callback")
	require.NoError(t, err)

	_, err = store.ExchangeAuthCode(context.Background(), code, "client-1", "http://evil.com/callback")
	assert.Error(t, err)
}

func TestExchangeAuthCodeDeletesAfterUse(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	code, err := store.CreateAuthCode(context.Background(), "user-1", "client-1", "http://localhost/callback")
	require.NoError(t, err)

	_, err = store.ExchangeAuthCode(context.Background(), code, "client-1", "http://localhost/callback")
	require.NoError(t, err)

	// Second exchange should fail — code is deleted
	_, err = store.ExchangeAuthCode(context.Background(), code, "client-1", "http://localhost/callback")
	assert.Error(t, err)
}

// --- Critical Session tests ---

func TestCreateCriticalSession(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	token, err := store.CreateCriticalSession(context.Background(), "user-1")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateCriticalSessionSuccess(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	token, err := store.CreateCriticalSession(context.Background(), "user-1")
	require.NoError(t, err)

	err = store.ValidateCriticalSession(context.Background(), token, "user-1")
	require.NoError(t, err)
}

func TestValidateCriticalSessionMismatch(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	token, err := store.CreateCriticalSession(context.Background(), "user-1")
	require.NoError(t, err)

	err = store.ValidateCriticalSession(context.Background(), token, "user-2")
	assert.ErrorIs(t, err, ErrSessionInvalid)
}

func TestValidateCriticalSessionNotFound(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	err := store.ValidateCriticalSession(context.Background(), "nonexistent", "user-1")
	assert.ErrorIs(t, err, ErrSessionExpired)
}

func TestValidateCriticalSessionDeletesAfterUse(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	token, err := store.CreateCriticalSession(context.Background(), "user-1")
	require.NoError(t, err)

	// First validation succeeds
	err = store.ValidateCriticalSession(context.Background(), token, "user-1")
	require.NoError(t, err)

	// Second should fail — session is deleted
	err = store.ValidateCriticalSession(context.Background(), token, "user-1")
	assert.ErrorIs(t, err, ErrSessionExpired)
}

func TestCriticalSessionExpiration(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	token, err := store.CreateCriticalSession(context.Background(), "user-1")
	require.NoError(t, err)

	// Fast-forward past 15-minute TTL
	mr.FastForward(16 * time.Minute)

	err = store.ValidateCriticalSession(context.Background(), token, "user-1")
	assert.ErrorIs(t, err, ErrSessionExpired)
}

// --- User Session tests ---

func TestAddAndGetUserSession(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	err := store.AddUserSession(context.Background(), "user-1", "token-abc", 10*time.Minute)
	require.NoError(t, err)

	token, err := store.GetUserSession(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, "token-abc", token)
}

func TestGetUserSessionNotFound(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	_, err := store.GetUserSession(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestInvalidateUserSession(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	err := store.AddUserSession(context.Background(), "user-1", "token-abc", 10*time.Minute)
	require.NoError(t, err)

	err = store.InvalidateUserSession(context.Background(), "user-1")
	require.NoError(t, err)

	_, err = store.GetUserSession(context.Background(), "user-1")
	assert.Error(t, err)
}

func TestInvalidateNonExistentSession(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	err := store.InvalidateUserSession(context.Background(), "nonexistent")
	assert.NoError(t, err) // should not error
}

func TestUserSessionExpiration(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	err := store.AddUserSession(context.Background(), "user-1", "token-abc", 1*time.Second)
	require.NoError(t, err)

	mr.FastForward(1100 * time.Millisecond)

	_, err = store.GetUserSession(context.Background(), "user-1")
	assert.Error(t, err)
}

// --- Error values tests ---

func TestErrorValues(t *testing.T) {
	assert.Contains(t, ErrCodeNotFound.Error(), "authorization code not found")
	assert.Contains(t, ErrCodeInvalid.Error(), "invalid authorization code")
	assert.Contains(t, ErrCodeMismatch.Error(), "client_id or redirect_uri mismatch")
	assert.Contains(t, ErrSessionExpired.Error(), "critical session expired")
	assert.Contains(t, ErrSessionInvalid.Error(), "invalid critical session")
}

func TestExchangeAuthCodeInvalidFormat(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	ctx := context.Background()
	client := store.client
	require.NoError(t, client.Set(ctx, "auth_code:invalid-no-pipes", "bad-format", 5*time.Minute))

	_, err := store.ExchangeAuthCode(ctx, "invalid-no-pipes", "client-1", "http://localhost/callback")
	assert.ErrorIs(t, err, ErrCodeInvalid)
}

func TestCreateAuthCode_StoreFailure(t *testing.T) {
	store, mr := setupSessionTest(t)
	mr.Close()

	_, err := store.CreateAuthCode(context.Background(), "user-1", "client-1", "http://localhost/callback")
	assert.Error(t, err)
}

func TestCreateCriticalSession_StoreFailure(t *testing.T) {
	store, mr := setupSessionTest(t)
	mr.Close()

	_, err := store.CreateCriticalSession(context.Background(), "user-1")
	assert.Error(t, err)
}

func TestExchangeAuthCode_DeleteFailure(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	code, err := store.CreateAuthCode(context.Background(), "user-1", "client-1", "http://localhost/callback")
	require.NoError(t, err)

	mr.Close()

	_, err = store.ExchangeAuthCode(context.Background(), code, "client-1", "http://localhost/callback")
	assert.Error(t, err)
}

func TestNewSessionStoreFromRedis(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewSessionStoreFromRedis(rdb)
	assert.NotNil(t, store)
}

func TestNewSessionStoreFromValkey(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewSessionStoreFromValkey(rdb)
	assert.NotNil(t, store)
}

type sessionErrorReader struct{}

func (sessionErrorReader) Read([]byte) (int, error) {
	return 0, errors.New("rand failed")
}

func TestGenerateCode_RandFailure(t *testing.T) {
	oldReader := rand.Reader
	defer func() { rand.Reader = oldReader }()
	rand.Reader = sessionErrorReader{}

	_, err := generateCode()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generate code")
}

func TestCreateAuthCode_GenerateCodeFailure(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	oldReader := rand.Reader
	defer func() { rand.Reader = oldReader }()
	rand.Reader = sessionErrorReader{}

	_, err := store.CreateAuthCode(context.Background(), "user-1", "client-1", "http://localhost/callback")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generate code")
}

func TestCreateCriticalSession_GenerateCodeFailure(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	oldReader := rand.Reader
	defer func() { rand.Reader = oldReader }()
	rand.Reader = sessionErrorReader{}

	_, err := store.CreateCriticalSession(context.Background(), "user-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generate code")
}

func TestExchangeAuthCode_DelFailure(t *testing.T) {
	store, mr := setupSessionTest(t)
	defer mr.Close()

	code, err := store.CreateAuthCode(context.Background(), "user-1", "client-1", "http://localhost/callback")
	require.NoError(t, err)

	failingStore := &SessionStore{
		client: &delFailingClient{
			inner: store.client,
		},
	}

	userID, err := failingStore.ExchangeAuthCode(context.Background(), code, "client-1", "http://localhost/callback")
	assert.Error(t, err)
	assert.Empty(t, userID)
}

type delFailingClient struct {
	inner CacheClient
}

func (m *delFailingClient) Get(ctx context.Context, key string) (string, error) {
	return m.inner.Get(ctx, key)
}

func (m *delFailingClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return m.inner.Set(ctx, key, value, expiration)
}

func (m *delFailingClient) Del(ctx context.Context, keys ...string) error {
	return errors.New("del failed")
}
