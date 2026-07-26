// Package claims provides domain types for JWT authentication.
// These types have no dependency on JWT libraries and can be used
// in any layer of the hexagonal architecture.
package claims

import (
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the domain model for JWT claims.
// It embeds jwt.RegisteredClaims because the JWT library requires it,
// but the domain layer only depends on the standard jwt/v5 types.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWKSKey represents a JSON Web Key in JWKS format.
type JWKSKey struct {
	KTY string `json:"kty"`
	CRV string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	KID string `json:"kid,omitempty"`
}

// JWKSResponse represents a JSON Web Key Set.
type JWKSResponse struct {
	Keys []JWKSKey `json:"keys"`
}
