// Package infra provides infrastructure implementations for the gateway service.
// This package depends on external libraries and should only be used in the composition root.
package infra

import (
	"time"

	"github.com/MAMUER/project/internal/auth/claims"
	"github.com/MAMUER/project/internal/auth/jwt"
)

// JWTAdapter implements the TokenProvider port using the internal JWT library.
// It is the bridge between the gateway's application layer and the JWT infrastructure.
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

// GenerateAccessToken implements ports.TokenProvider.
func (a *JWTAdapter) GenerateAccessToken(userID, email, role string, ttl time.Duration) (string, error) {
	return jwt.GenerateAccessToken(userID, email, role, a.privateKeyPEM, ttl)
}

// GenerateRefreshToken implements ports.TokenProvider.
func (a *JWTAdapter) GenerateRefreshToken() string {
	return jwt.GenerateRefreshToken()
}

// ValidateAccessToken implements ports.TokenProvider.
func (a *JWTAdapter) ValidateAccessToken(token string) (*claims.Claims, error) {
	return jwt.ValidateAccessToken(token, a.publicKeyPEM)
}

// ComputeTokenFingerprint implements ports.TokenProvider.
func (a *JWTAdapter) ComputeTokenFingerprint(token string) string {
	return jwt.ComputeTokenFingerprint(token)
}

// PublicKeyPEMToJWKS implements ports.TokenProvider.
func (a *JWTAdapter) PublicKeyPEMToJWKS(publicKeyPEM string) ([]byte, error) {
	return jwt.PublicKeyPEMToJWKS(publicKeyPEM)
}

// PublicKeyPEM implements ports.TokenProvider.
func (a *JWTAdapter) PublicKeyPEM() string {
	return a.publicKeyPEM
}
