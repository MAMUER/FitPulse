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
