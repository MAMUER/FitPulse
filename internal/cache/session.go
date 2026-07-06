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

// SessionStore управляет сессиями и однократными кодами авторизации
// Требование #2: Однократное использование токенов
// Требование #3: Разделение хранилищ сессий
type SessionStore struct {
	client *Client // redis client
}

// NewSessionStore создаёт хранилище сессий
func NewSessionStore(client *Client) *SessionStore {
	return &SessionStore{client: client}
}

// NewSessionStoreFromRedis создаёт хранилище сессий из существующего Redis клиента
func NewSessionStoreFromRedis(rdb *redis.Client) *SessionStore {
	return &SessionStore{client: NewClientFromRedis(rdb)}
}

// generateCode генерирует криптографически безопасный код
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

	// Код живёт 5 минут и удаётся после использования
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

	// Парсим сохранённые данные
	parts := strings.SplitN(value, "|", 3)
	if len(parts) != 3 {
		return "", ErrCodeInvalid
	}
	savedUserID, savedClientID, savedRedirectURI := parts[0], parts[1], parts[2]

	// Требование #2: Проверяем client_id и redirect_uri
	if savedClientID != clientID || savedRedirectURI != redirectURI {
		return "", ErrCodeMismatch
	}

	// Требование #2: Удаляем код — он больше недействителен
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

	// Требование #3: Сессия удаляется после использования
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
