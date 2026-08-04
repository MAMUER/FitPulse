// Package jwt provides infrastructure utilities for JWT token generation,
// parsing, JWKS conversion, and a shared JWTAdapter implementation.
package jwt

import (
	"time"

	"github.com/MAMUER/project/internal/auth/claims"
)

// JWTAdapter implements the TokenProvider port using the internal JWT library.
// It is the bridge between the application layer and the JWT infrastructure.
type JWTAdapter struct {
	privateKeyPEM string
	publicKeyPEM  string
}

// NewJWTAdapter creates a new JWTAdapter with the given key pair.
func NewJWTAdapter(privateKeyPEM, publicKeyPEM string) *JWTAdapter {
	return &JWTAdapter{
		privateKeyPEM: privateKeyPEM,
		publicKeyPEM:  publicKeyPEM,
	}
}

// GenerateAccessToken creates a signed ES256 JWT access token.
func (a *JWTAdapter) GenerateAccessToken(userID, email, role string, ttl time.Duration) (string, error) {
	return GenerateAccessToken(userID, email, role, a.privateKeyPEM, ttl)
}

// GenerateRefreshToken creates a cryptographically secure refresh token.
func (a *JWTAdapter) GenerateRefreshToken() string {
	return GenerateRefreshToken()
}

// ValidateAccessToken parses and validates an ES256 JWT access token.
func (a *JWTAdapter) ValidateAccessToken(token string) (*claims.Claims, error) {
	return ValidateAccessToken(token, a.publicKeyPEM)
}

// ComputeTokenFingerprint computes a SHA256 fingerprint of a token string.
func (a *JWTAdapter) ComputeTokenFingerprint(token string) string {
	return ComputeTokenFingerprint(token)
}

// PublicKeyPEMToJWKS converts an EC P-256 public key PEM to JWKS JSON.
func (a *JWTAdapter) PublicKeyPEMToJWKS(publicKeyPEM string) ([]byte, error) {
	return PublicKeyPEMToJWKS(publicKeyPEM)
}

// PublicKeyPEM returns the EC P-256 public key PEM used for token validation.
func (a *JWTAdapter) PublicKeyPEM() string {
	return a.publicKeyPEM
}
