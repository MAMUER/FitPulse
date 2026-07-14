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

// SessionStore manages sessions and one-time authorization codes
type SessionStore struct {
	client *Client // valkey client
}

// NewSessionStore creates a session store
func NewSessionStore(client *Client) *SessionStore {
	return &SessionStore{client: client}
}

// NewSessionStoreFromValkey creates a session store from an existing Valkey client
func NewSessionStoreFromValkey(rdb *redis.Client) *SessionStore {
	return &SessionStore{client: NewClientFromValkey(rdb)}
}

// generateCode generates a cryptographically secure code
func generateCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b)[:43], nil
}

func (s *SessionStore) CreateAuthCode(ctx context.Context, userID, clientID, redirectURI string) (string, error) {
	code, err := generateCode()
	if err != nil {
		return "", err
	}
	key := "auth_code:" + code
	value := fmt.Sprintf("%s|%s|%s", userID, clientID, redirectURI)

	// Code lives 5 minutes and is deleted after use
	if err := s.client.Set(ctx, key, value, 5*time.Minute); err != nil {
		return "", err
	}

	return code, nil
}

func (s *SessionStore) ExchangeAuthCode(ctx context.Context, code, clientID, redirectURI string) (string, error) {
	key := "auth_code:" + code

	value, err := s.client.Get(ctx, key)
	if err != nil {
		return "", ErrCodeNotFound
	}

	// Parse saved data
	parts := strings.SplitN(value, "|", 3)
	if len(parts) != 3 {
		return "", ErrCodeInvalid
	}
	savedUserID, savedClientID, savedRedirectURI := parts[0], parts[1], parts[2]

	// Requirement #2: Verify client_id and redirect_uri
	if savedClientID != clientID || savedRedirectURI != redirectURI {
		return "", ErrCodeMismatch
	}

	// Requirement #2: Delete code — it's no longer valid
	if err := s.client.Del(ctx, key); err != nil {
		return "", err
	}

	return savedUserID, nil
}

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

func (s *SessionStore) ValidateCriticalSession(ctx context.Context, token, expectedUserID string) error {
	key := "critical_session:" + token

	userID, err := s.client.Get(ctx, key)
	if err != nil {
		return ErrSessionExpired
	}

	if userID != expectedUserID {
		return ErrSessionInvalid
	}

	// Requirement #3: Session is deleted after use
	return s.client.Del(ctx, key)
}

func (s *SessionStore) InvalidateUserSession(ctx context.Context, userID string) error {
	key := "user_sessions:" + userID
	return s.client.Del(ctx, key)
}

func (s *SessionStore) AddUserSession(ctx context.Context, userID, sessionToken string, ttl time.Duration) error {
	key := "user_sessions:" + userID
	return s.client.Set(ctx, key, sessionToken, ttl)
}

func (s *SessionStore) GetUserSession(ctx context.Context, userID string) (string, error) {
	key := "user_sessions:" + userID
	val, err := s.client.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return val, nil
}

var (
	ErrCodeNotFound   = errors.New("authorization code not found or already used")
	ErrCodeInvalid    = errors.New("invalid authorization code")
	ErrCodeMismatch   = errors.New("client_id or redirect_uri mismatch")
	ErrSessionExpired = errors.New("critical session expired")
	ErrSessionInvalid = errors.New("invalid critical session")
)
