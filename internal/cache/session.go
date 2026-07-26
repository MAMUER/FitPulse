// Package cache provides caching utilities for user sessions and data.
package cache

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionStore manages sessions and one-time authorization codes in Valkey.
type SessionStore struct {
	client *Client
}

// NewSessionStore creates a session store with the given cache client.
func NewSessionStore(client *Client) *SessionStore {
	return &SessionStore{client: client}
}

// NewSessionStoreFromRedis creates a session store from an existing Redis/Valkey client.
// Valkey is wire-compatible with Redis, so this works for Valkey clients as well.
func NewSessionStoreFromRedis(rdb *redis.Client) *SessionStore {
	return &SessionStore{client: FromRedisClient(rdb)}
}

// NewSessionStoreFromValkey creates a session store from an existing Valkey client.
// Deprecated: use NewSessionStoreFromRedis instead.
func NewSessionStoreFromValkey(rdb *redis.Client) *SessionStore {
	return NewSessionStoreFromRedis(rdb)
}

// generateCode generates a cryptographically secure authorization code.
func generateCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b)[:43], nil
}

// CreateAuthCode creates a new OAuth2 authorization code and stores it in Valkey.
// The code expires after 5 minutes and is deleted after successful exchange.
func (s *SessionStore) CreateAuthCode(ctx context.Context, userID, clientID, redirectURI string) (string, error) {
	code, err := generateCode()
	if err != nil {
		return "", err
	}
	key := "auth_code:" + code
	value := fmt.Sprintf("%s|%s|%s", userID, clientID, redirectURI)

	if err := s.client.Set(ctx, key, value, 5*time.Minute); err != nil {
		return "", err
	}

	return code, nil
}

// ExchangeAuthCode exchanges an authorization code for a user ID.
// It verifies client_id and redirect_uri, then deletes the code to prevent reuse.
func (s *SessionStore) ExchangeAuthCode(ctx context.Context, code, clientID, redirectURI string) (string, error) {
	key := "auth_code:" + code

	value, err := s.client.Get(ctx, key)
	if err != nil {
		return "", ErrCodeNotFound
	}

	parts := strings.SplitN(value, "|", 3)
	if len(parts) != 3 {
		return "", ErrCodeInvalid
	}
	savedUserID, savedClientID, savedRedirectURI := parts[0], parts[1], parts[2]

	if savedClientID != clientID || savedRedirectURI != redirectURI {
		return "", ErrCodeMismatch
	}

	if err := s.client.Del(ctx, key); err != nil {
		return "", err
	}

	return savedUserID, nil
}

// CreateCriticalSession creates a one-time critical session token in Valkey.
// The session expires after 15 minutes and is deleted after successful validation.
func (s *SessionStore) CreateCriticalSession(ctx context.Context, userID string) (string, error) {
	token, err := generateCode()
	if err != nil {
		return "", err
	}
	key := "critical_session:" + token
	if err := s.client.Set(ctx, key, userID, 15*time.Minute); err != nil {
		return "", err
	}

	return token, nil
}

// ValidateCriticalSession validates a critical session token.
// It checks the user ID and deletes the session to prevent reuse.
func (s *SessionStore) ValidateCriticalSession(ctx context.Context, token, expectedUserID string) error {
	key := "critical_session:" + token

	userID, err := s.client.Get(ctx, key)
	if err != nil {
		return ErrSessionExpired
	}

	if userID != expectedUserID {
		return ErrSessionInvalid
	}

	return s.client.Del(ctx, key)
}

// InvalidateUserSession invalidates all sessions for a user.
func (s *SessionStore) InvalidateUserSession(ctx context.Context, userID string) error {
	key := "user_sessions:" + userID
	return s.client.Del(ctx, key)
}

// AddUserSession adds a session token for a user.
func (s *SessionStore) AddUserSession(ctx context.Context, userID, sessionToken string, ttl time.Duration) error {
	key := "user_sessions:" + userID
	return s.client.Set(ctx, key, sessionToken, ttl)
}

// GetUserSession retrieves the session token for a user.
func (s *SessionStore) GetUserSession(ctx context.Context, userID string) (string, error) {
	key := "user_sessions:" + userID
	val, err := s.client.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return val, nil
}

var (
	// ErrCodeNotFound is returned when an authorization code is not found or already used.
	ErrCodeNotFound = errors.New("authorization code not found or already used")
	// ErrCodeInvalid is returned when an authorization code is malformed.
	ErrCodeInvalid = errors.New("invalid authorization code")
	// ErrCodeMismatch is returned when client_id or redirect_uri does not match.
	ErrCodeMismatch = errors.New("client_id or redirect_uri mismatch")
	// ErrSessionExpired is returned when a critical session is not found or expired.
	ErrSessionExpired = errors.New("critical session expired")
	// ErrSessionInvalid is returned when a critical session user ID does not match.
	ErrSessionInvalid = errors.New("invalid critical session")
)
