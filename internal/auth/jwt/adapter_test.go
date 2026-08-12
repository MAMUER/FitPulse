package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/auth/claims"
)

func generateAdapterTestKeyPair() (string, string) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privateKeyBytes, _ := x509.MarshalECPrivateKey(privateKey)
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}))
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}))
	return privateKeyPEM, publicKeyPEM
}

func TestNewJWTAdapter(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateAdapterTestKeyPair()
	adapter := NewJWTAdapter(privateKeyPEM, publicKeyPEM)
	assert.NotNil(t, adapter)
	assert.Equal(t, privateKeyPEM, adapter.privateKeyPEM)
	assert.Equal(t, publicKeyPEM, adapter.publicKeyPEM)
}

func TestJWTAdapter_GenerateAccessToken(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateAdapterTestKeyPair()
	adapter := NewJWTAdapter(privateKeyPEM, publicKeyPEM)

	token, err := adapter.GenerateAccessToken("user-123", "test@example.com", "client", 15*time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := adapter.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "client", claims.Role)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), claims.ExpiresAt.Time, 2*time.Second)
}

func TestJWTAdapter_GenerateRefreshToken(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateAdapterTestKeyPair()
	adapter := NewJWTAdapter(privateKeyPEM, publicKeyPEM)

	token := adapter.GenerateRefreshToken()
	assert.NotEmpty(t, token)
	assert.Len(t, token, 43)
}

func TestJWTAdapter_ValidateAccessToken(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateAdapterTestKeyPair()
	adapter := NewJWTAdapter(privateKeyPEM, publicKeyPEM)

	token, err := GenerateAccessToken("user-123", "test@example.com", "client", privateKeyPEM, 15*time.Minute)
	require.NoError(t, err)

	claims, err := adapter.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
}

func TestJWTAdapter_ValidateAccessToken_Invalid(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateAdapterTestKeyPair()
	adapter := NewJWTAdapter(privateKeyPEM, publicKeyPEM)

	_, err := adapter.ValidateAccessToken("invalid.token")
	assert.Error(t, err)
}

func TestJWTAdapter_ComputeTokenFingerprint(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateAdapterTestKeyPair()
	adapter := NewJWTAdapter(privateKeyPEM, publicKeyPEM)

	fp1 := adapter.ComputeTokenFingerprint("token-abc")
	fp2 := adapter.ComputeTokenFingerprint("token-abc")
	fp3 := adapter.ComputeTokenFingerprint("token-def")

	assert.NotEmpty(t, fp1)
	assert.Equal(t, fp1, fp2)
	assert.NotEqual(t, fp1, fp3)
}

func TestJWTAdapter_PublicKeyPEMToJWKS(t *testing.T) {
	_, publicKeyPEM := generateAdapterTestKeyPair()
	adapter := NewJWTAdapter("", publicKeyPEM)

	jsonBytes, err := adapter.PublicKeyPEMToJWKS(publicKeyPEM)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	var jwks claims.JWKSResponse
	err = json.Unmarshal(jsonBytes, &jwks)
	require.NoError(t, err)
	require.Len(t, jwks.Keys, 1)
	assert.Equal(t, "EC", jwks.Keys[0].KTY)
	assert.Equal(t, "P-256", jwks.Keys[0].CRV)
	assert.NotEmpty(t, jwks.Keys[0].X)
	assert.NotEmpty(t, jwks.Keys[0].Y)
}

func TestJWTAdapter_PublicKeyPEMToJWKS_EmptyPEM(t *testing.T) {
	adapter := NewJWTAdapter("", "")
	_, err := adapter.PublicKeyPEMToJWKS("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "public key PEM cannot be empty")
}

func TestJWTAdapter_PublicKeyPEM(t *testing.T) {
	_, publicKeyPEM := generateAdapterTestKeyPair()
	adapter := NewJWTAdapter("", publicKeyPEM)
	assert.Equal(t, publicKeyPEM, adapter.PublicKeyPEM())
}

func TestJWTAdapter_PublicKeyPEMToJWKS_Parse(t *testing.T) {
	var target claims.JWKSResponse
	err := json.Unmarshal([]byte(`{"keys":[{"kty":"EC","crv":"P-256","x":"test","y":"test2"}]}`), &target)
	assert.NoError(t, err)
	assert.Len(t, target.Keys, 1)
	assert.Equal(t, "EC", target.Keys[0].KTY)
}
