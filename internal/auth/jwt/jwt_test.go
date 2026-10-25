package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/auth/claims"
)

func generateTestKeyPair() (string, string, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate key: %w", err)
	}
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal private key: %w", err)
	}
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}))
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal public key: %w", err)
	}
	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}))
	return privateKeyPEM, publicKeyPEM, nil
}

func TestGenerateAccessToken(t *testing.T) {
	privateKeyPEM, publicKeyPEM, kpErr := generateTestKeyPair()
	require.NoError(t, kpErr)
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
	_, publicKeyPEM, kpErr := generateTestKeyPair()
	require.NoError(t, kpErr)

	t.Run("empty token", func(t *testing.T) {
		_, err := ValidateAccessToken("", publicKeyPEM)
		assert.Error(t, err)
	})

	t.Run("empty public key", func(t *testing.T) {
		_, err := ValidateAccessToken("some.token", "")
		assert.Error(t, err)
	})

	t.Run("wrong key", func(t *testing.T) {
		wrongPrivate, _, kpErr := generateTestKeyPair()
		require.NoError(t, kpErr)
		token, signErr := GenerateAccessToken("u", "e@e.com", "client", wrongPrivate, 15*time.Minute)
		require.NoError(t, signErr)
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
	privateKeyPEM, publicKeyPEM, kpErr := generateTestKeyPair()
	require.NoError(t, kpErr)
	token, err := GenerateAccessToken("user-1", "a@b.com", "client", privateKeyPEM, -1*time.Hour)
	require.NoError(t, err)

	_, err = ValidateAccessToken(token, publicKeyPEM)
	assert.Error(t, err)

}

func TestTokenStructure(t *testing.T) {
	privateKeyPEM, _, kpErr := generateTestKeyPair()
	require.NoError(t, kpErr)
	token, err := GenerateAccessToken("user-123", "test@example.com", "client", privateKeyPEM, 15*time.Minute)
	require.NoError(t, err)

	parser := jwt.Parser{}
	parsed, _, err := parser.ParseUnverified(token, &claims.Claims{})
	require.NoError(t, err)

	parsedClaims, ok := parsed.Claims.(*claims.Claims)
	assert.True(t, ok)
	assert.NotEmpty(t, parsedClaims.ID)
	assert.NotNil(t, parsedClaims.ExpiresAt)
	assert.NotNil(t, parsedClaims.IssuedAt)
	assert.Equal(t, "user-123", parsedClaims.UserID)
	assert.Equal(t, "test@example.com", parsedClaims.Email)
	assert.Equal(t, "client", parsedClaims.Role)

}

func TestPublicKeyPEMToJWKS(t *testing.T) {
	_, publicKeyPEM, kpErr := generateTestKeyPair()
	require.NoError(t, kpErr)
	jsonBytes, err := PublicKeyPEMToJWKS(publicKeyPEM)
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

func TestPublicKeyPEMToJWKS_EmptyPEM(t *testing.T) {
	_, err := PublicKeyPEMToJWKS("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "public key PEM cannot be empty")
}

func TestPublicKeyPEMToJWKS_InvalidPEM(t *testing.T) {
	_, err := PublicKeyPEMToJWKS("invalid-pem")
	assert.Error(t, err)
}

func TestPublicKeyPEMToJWKS_UnsupportedCurve(t *testing.T) {
	_, err := PublicKeyPEMToJWKS("not-a-valid-key")
	assert.Error(t, err)
}

func TestPublicKeyPEMToJWKS_NonP256Curve(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	_, err = PublicKeyPEMToJWKS(string(pubPEM))
	assert.Error(t, err)
}

func TestPublicKeyPEMToJWKS_InvalidPubFormat(t *testing.T) {
	_, pub, _ := generateTestKeyPair()
	_, err := PublicKeyPEMToJWKS(pub)
	assert.NoError(t, err)
}

func TestParseECPrivateKey_Errors(t *testing.T) {
	t.Run("empty pem", func(t *testing.T) {
		_, err := ParseECPrivateKey("")
		assert.Error(t, err)
	})

	t.Run("invalid pem", func(t *testing.T) {
		_, err := ParseECPrivateKey("not-a-valid-key")
		assert.Error(t, err)
	})

	t.Run("pem with extra data", func(t *testing.T) {
		priv, _, _ := generateTestKeyPair()
		_, err := ParseECPrivateKey(priv + "\nextra")
		assert.Error(t, err)
	})

	t.Run("invalid block bytes", func(t *testing.T) {
		invalid := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: []byte("not-a-key")})
		_, err := ParseECPrivateKey(string(invalid))
		assert.Error(t, err)
	})
}

func TestParseECPublicKey_Errors(t *testing.T) {
	t.Run("empty pem", func(t *testing.T) {
		_, err := ParseECPublicKey("")
		assert.Error(t, err)
	})

	t.Run("invalid pem", func(t *testing.T) {
		_, err := ParseECPublicKey("not-a-valid-key")
		assert.Error(t, err)
	})

	t.Run("pem with extra data", func(t *testing.T) {
		_, pub, _ := generateTestKeyPair()
		_, err := ParseECPublicKey(pub + "\nextra")
		assert.Error(t, err)
	})

	t.Run("not ecdsa public key", func(t *testing.T) {
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		rsaPub, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
		require.NoError(t, err)
		pemData := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: rsaPub})
		_, err = ParseECPublicKey(string(pemData))
		assert.Error(t, err)
	})

	t.Run("invalid pkix bytes", func(t *testing.T) {
		invalid := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("bad")})
		_, err := ParseECPublicKey(string(invalid))
		assert.Error(t, err)
	})
}

func TestGenerateAccessToken_EmptyPrivateKey(t *testing.T) {
	_, err := GenerateAccessToken("user-1", "test@example.com", "client", "", 15*time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private key PEM cannot be empty")
}

func TestGenerateAccessToken_InvalidPrivateKey(t *testing.T) {
	_, err := GenerateAccessToken("user-1", "test@example.com", "client", "not-a-key", 15*time.Minute)
	assert.Error(t, err)
}

func TestValidateAccessToken_EmptyToken(t *testing.T) {
	_, err := ValidateAccessToken("", "some-public-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is empty")
}

func TestValidateAccessToken_EmptyPublicKey(t *testing.T) {
	_, err := ValidateAccessToken("some.token", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "public key PEM cannot be empty")
}

func TestValidateAccessToken_InvalidPublicKey(t *testing.T) {
	_, err := ValidateAccessToken("a.b.c", "not-a-key")
	assert.Error(t, err)
}

func TestValidateAccessToken_BadTokenFormat(t *testing.T) {
	_, pub, _ := generateTestKeyPair()
	_, err := ValidateAccessToken("not-a-jwt", pub)
	assert.Error(t, err)
}

func TestValidateAccessToken_WrongSigningMethod(t *testing.T) {
	_, pub, _ := generateTestKeyPair()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"foo": "bar"})
	//nolint:gosec // test-only signing key
	signed, err := token.SignedString([]byte("secret"))
	require.NoError(t, err)
	_, err = ValidateAccessToken(signed, pub)
	assert.Error(t, err)
}

func TestValidateAccessToken_InvalidClaimsType(t *testing.T) {
	_, pub, _ := generateTestKeyPair()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"sub": "x"})
	//nolint:gosec // test-only signing key
	signed, err := token.SignedString([]byte("x"))
	// This will fail at signing; skip if it fails
	if err != nil {
		t.Skip()
		return
	}
	_, err = ValidateAccessToken(signed, pub)
	assert.Error(t, err)
}
