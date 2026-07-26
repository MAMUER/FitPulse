// Package jwt provides infrastructure utilities for JWT token generation,
// parsing, and JWKS conversion. It depends on the domain types from
// `internal/auth/claims` and should only be used in the infrastructure layer.
package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/MAMUER/project/internal/auth/claims"
)

// GenerateES256KeyPair generates an ES256 (P-256) key pair and returns PEM-encoded strings.
func GenerateES256KeyPair() (string, string, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate ECDSA key: %w", err)
	}
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal EC private key: %w", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal PKIX public key: %w", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return string(privateKeyPEM), string(publicKeyPEM), nil
}

// ParseECPrivateKey parses an EC private key from PEM format.
func ParseECPrivateKey(privateKeyPEM string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing EC private key")
	}
	privKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse EC private key: %w", err)
	}
	return privKey, nil
}

// ParseECPublicKey parses an EC public key from PEM format.
func ParseECPublicKey(publicKeyPEM string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing public key")
	}
	pub, parseErr := x509.ParsePKIXPublicKey(block.Bytes)
	if parseErr != nil {
		return nil, fmt.Errorf("parse PKIX public key: %w", parseErr)
	}
	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("not an ECDSA public key")
	}
	return ecdsaPub, nil
}

// GenerateAccessToken creates a signed ES256 JWT access token.
func GenerateAccessToken(userID, email, role, privateKeyPEM string, ttl time.Duration) (string, error) {
	if privateKeyPEM == "" {
		return "", errors.New("private key PEM cannot be empty")
	}
	privateKey, err := ParseECPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}

	now := time.Now()
	c := claims.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, c)
	signedToken, signErr := token.SignedString(privateKey)
	if signErr != nil {
		return "", fmt.Errorf("sign access token: %w", signErr)
	}
	return signedToken, nil
}

// GenerateRefreshToken creates a cryptographically secure refresh token.
func GenerateRefreshToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// ValidateAccessToken parses and validates an ES256 JWT access token.
func ValidateAccessToken(tokenString, publicKeyPEM string) (*claims.Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}
	if publicKeyPEM == "" {
		return nil, errors.New("public key PEM cannot be empty")
	}

	publicKey, err := ParseECPublicKey(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse EC public key: %w", err)
	}

	token, parseErr := jwt.ParseWithClaims(tokenString, &claims.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return publicKey, nil
	})
	if parseErr != nil {
		return nil, fmt.Errorf("parse access token: %w", parseErr)
	}
	if c, ok := token.Claims.(*claims.Claims); ok && token.Valid {
		return c, nil
	}
	return nil, errors.New("invalid token")
}

// ComputeTokenFingerprint computes a SHA256 fingerprint of a token string.
func ComputeTokenFingerprint(tokenString string) string {
	hash := sha256.Sum256([]byte(tokenString))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// PublicKeyPEMToJWKS converts an EC P-256 public key PEM to JWKS JSON.
func PublicKeyPEMToJWKS(publicKeyPEM string) ([]byte, error) {
	if publicKeyPEM == "" {
		return nil, errors.New("public key PEM cannot be empty")
	}

	publicKey, err := ParseECPublicKey(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse EC public key for JWKS: %w", err)
	}

	if publicKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf("unsupported curve: %v", publicKey.Curve)
	}

	pubBytes, err := publicKey.Bytes()
	if err != nil {
		return nil, fmt.Errorf("encode public key: %w", err)
	}
	if len(pubBytes) != 65 || pubBytes[0] != 0x04 {
		return nil, errors.New("unexpected public key format")
	}
	xBytes := pubBytes[1:33]
	yBytes := pubBytes[33:65]

	key := claims.JWKSKey{
		KTY: "EC",
		CRV: "P-256",
		X:   base64.RawURLEncoding.EncodeToString(xBytes),
		Y:   base64.RawURLEncoding.EncodeToString(yBytes),
	}

	resp := claims.JWKSResponse{Keys: []claims.JWKSKey{key}}
	return json.Marshal(resp)
}
