// Package ports defines the authentication port for the gateway service.
// This is the interface that the gateway's application layer depends on.
// The actual implementation is provided by the infrastructure layer (infra/jwt_adapter.go).
package ports

import (
	"time"

	"github.com/MAMUER/project/internal/auth/claims"
)

// TokenProvider is the authentication port for the gateway service.
// It provides JWT token generation and validation capabilities.
type TokenProvider interface {
	// GenerateAccessToken creates a signed ES256 JWT access token.
	GenerateAccessToken(userID, email, role string, ttl time.Duration) (string, error)

	// GenerateRefreshToken creates a cryptographically secure refresh token.
	GenerateRefreshToken() string

	// ValidateAccessToken parses and validates an ES256 JWT access token.
	ValidateAccessToken(token string) (*claims.Claims, error)

	// ComputeTokenFingerprint computes a SHA256 fingerprint of a token string.
	ComputeTokenFingerprint(token string) string

	// PublicKeyPEMToJWKS converts an EC P-256 public key PEM to JWKS JSON.
	PublicKeyPEMToJWKS(publicKeyPEM string) ([]byte, error)

	// PublicKeyPEM returns the EC P-256 public key PEM used for token validation.
	PublicKeyPEM() string
}
