package auth

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
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

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

func GenerateAccessToken(userID, email, role, privateKeyPEM string, ttl time.Duration) (string, error) {
	if privateKeyPEM == "" {
		return "", errors.New("private key PEM cannot be empty")
	}
	privateKey, err := ParseECPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signedToken, signErr := token.SignedString(privateKey)
	if signErr != nil {
		return "", fmt.Errorf("sign access token: %w", signErr)
	}
	return signedToken, nil
}

func GenerateRefreshToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func ValidateAccessToken(tokenString, publicKeyPEM string) (*Claims, error) {
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

	token, parseErr := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return publicKey, nil
	})
	if parseErr != nil {
		return nil, fmt.Errorf("parse access token: %w", parseErr)
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func ComputeTokenFingerprint(tokenString string) string {
	hash := sha256.Sum256([]byte(tokenString))
	return base64.RawURLEncoding.EncodeToString(hash[:])
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

	xBytes := publicKey.X.Bytes()
	yBytes := publicKey.Y.Bytes()

	key := JWKSKey{
		KTY: "EC",
		CRV: "P-256",
		X:   base64.RawURLEncoding.EncodeToString(xBytes),
		Y:   base64.RawURLEncoding.EncodeToString(yBytes),
	}

	resp := JWKSResponse{Keys: []JWKSKey{key}}
	return json.Marshal(resp)
}
