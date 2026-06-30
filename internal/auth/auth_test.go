package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestKeyPair() (string, string) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privateKeyBytes, _ := x509.MarshalECPrivateKey(privateKey)
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}))
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}))
	return privateKeyPEM, publicKeyPEM
}

func TestGenerateAccessToken(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateTestKeyPair()
	token, err := GenerateAccessToken("user-123", "test@example.com", "client", privateKeyPEM, 15*time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ValidateAccessToken(token, publicKeyPEM)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "client", claims.Role)
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), claims.ExpiresAt.Time, 2*time.Second)
}

func TestValidateAccessToken_Errors(t *testing.T) {
	_, publicKeyPEM := generateTestKeyPair()

	t.Run("empty token", func(t *testing.T) {
		_, err := ValidateAccessToken("", publicKeyPEM)
		assert.Error(t, err)
	})

	t.Run("empty public key", func(t *testing.T) {
		_, err := ValidateAccessToken("some.token", "")
		assert.Error(t, err)
	})

	t.Run("wrong key", func(t *testing.T) {
		wrongPrivate, _ := generateTestKeyPair()
		token, _ := GenerateAccessToken("u", "e@e.com", "client", wrongPrivate, 15*time.Minute)
		_, err := ValidateAccessToken(token, publicKeyPEM)
		assert.Error(t, err)
	})
}

func TestGenerateRefreshToken(t *testing.T) {
	token := GenerateRefreshToken()
	assert.NotEmpty(t, token)
	assert.Len(t, token, 43)
}

func TestComputeTokenFingerprint(t *testing.T) {
	fp1 := ComputeTokenFingerprint("token-abc")
	fp2 := ComputeTokenFingerprint("token-abc")
	fp3 := ComputeTokenFingerprint("token-def")
	assert.NotEmpty(t, fp1)
	assert.Equal(t, fp1, fp2)
	assert.NotEqual(t, fp1, fp3)
}

func TestGenerateES256KeyPair(t *testing.T) {
	privatePEM, publicPEM, err := GenerateES256KeyPair()
	require.NoError(t, err)
	assert.NotEmpty(t, privatePEM)
	assert.NotEmpty(t, publicPEM)
	assert.Contains(t, privatePEM, "EC PRIVATE KEY")
	assert.Contains(t, publicPEM, "PUBLIC KEY")

	_, err = ParseECPrivateKey(privatePEM)
	assert.NoError(t, err)
	_, err = ParseECPublicKey(publicPEM)
	assert.NoError(t, err)
}

func TestExpiredAccessToken(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateTestKeyPair()
	token, err := GenerateAccessToken("user-1", "a@b.com", "client", privateKeyPEM, -1*time.Hour)
	require.NoError(t, err)

	_, err = ValidateAccessToken(token, publicKeyPEM)
	assert.Error(t, err)
	_ = publicKeyPEM
}

func TestTokenStructure(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateTestKeyPair()
	token, err := GenerateAccessToken("user-123", "test@example.com", "client", privateKeyPEM, 15*time.Minute)
	require.NoError(t, err)

	parser := jwt.Parser{}
	parsed, _, err := parser.ParseUnverified(token, &Claims{})
	require.NoError(t, err)

	claims, ok := parsed.Claims.(*Claims)
	assert.True(t, ok)
	assert.NotEmpty(t, claims.ID)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "client", claims.Role)
	_ = publicKeyPEM
}
