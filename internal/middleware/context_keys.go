// Package middleware provides HTTP middleware for authentication, logging, and security.
package middleware

// ContextKey is the type for the keys stored in context.Context. Using a
// dedicated type instead of the built-in string prevents collisions between
// keys defined in different packages.
type ContextKey string

const (
	// CorrelationIDKey - ключ для корреляционного идентификатора
	CorrelationIDKey ContextKey = "correlation_id"
	// UserIDKey - ключ для ID пользователя
	UserIDKey ContextKey = "user_id"
	// RoleKey - ключ для роли пользователя
	RoleKey ContextKey = "role"
	// RequestIDKey - ключ для идентификатора запроса
	RequestIDKey ContextKey = "request_id"
)
