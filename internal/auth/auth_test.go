package auth

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-auth-tests"

func TestGenerateJWT(t *testing.T) {
	tests := []struct {
		name            string
		userID          string
		email           string
		role            string
		secret          string
		expirationHours int
		wantErr         bool
		wantRole        string
	}{
		{
			name:            "valid token for client",
			userID:          "client-123",
			email:           "client@example.com",
			role:            "client",
			secret:          "mysecretkey123",
			expirationHours: 24,
			wantErr:         false,
			wantRole:        "client",
		},
		{
			name:            "valid token for admin",
			userID:          "admin-456",
			email:           "admin@example.com",
			role:            "admin",
			secret:          "mysecretkey123",
			expirationHours: 48,
			wantErr:         false,
			wantRole:        "admin",
		},
		{
			name:            "empty secret - should fail",
			userID:          "123",
			email:           "test@example.com",
			role:            "client",
			secret:          "",
			expirationHours: 24,
			wantErr:         true,
			wantRole:        "",
		},
		{
			name:            "zero expiration hours - uses default",
			userID:          "123",
			email:           "test@example.com",
			role:            "client",
			secret:          "mysecret",
			expirationHours: 0,
			wantErr:         false,
			wantRole:        "client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateJWT(tt.userID, tt.email, tt.role, tt.secret, tt.expirationHours)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				claims, err := ValidateJWT(token, tt.secret)
				assert.NoError(t, err)
				assert.Equal(t, tt.userID, claims.UserID)
				assert.Equal(t, tt.email, claims.Email)
				assert.Equal(t, tt.role, claims.Role)
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	secret := "test-secret-key-123"
	userID := "user-123"
	email := "user@example.com"
	role := "client"

	validToken, err := GenerateJWT(userID, email, role, secret, 24)
	require.NoError(t, err)
	require.NotEmpty(t, validToken)

	tests := []struct {
		name       string
		token      string
		secret     string
		wantErr    bool
		wantUserID string
		wantEmail  string
		wantRole   string
	}{
		{
			name:       "valid token",
			token:      validToken,
			secret:     secret,
			wantErr:    false,
			wantUserID: userID,
			wantEmail:  email,
			wantRole:   role,
		},
		{
			name:    "invalid secret",
			token:   validToken,
			secret:  "wrong-secret",
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "invalid.token.string",
			secret:  secret,
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			secret:  secret,
			wantErr: true,
		},
		{
			name:    "empty secret",
			token:   validToken,
			secret:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateJWT(tt.token, tt.secret)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, tt.wantUserID, claims.UserID)
				assert.Equal(t, tt.wantEmail, claims.Email)
				assert.Equal(t, tt.wantRole, claims.Role)
			}
		})
	}
}

func TestExpiredToken(t *testing.T) {
	secret := testSecret

	claims := Claims{
		UserID: "123",
		Email:  "test@example.com",
		Role:   "client",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ID:        "token-id",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	claimsResult, err := ValidateJWT(expiredToken, secret)
	assert.Error(t, err)
	assert.Nil(t, claimsResult)
}

func TestTokenWithFutureIssuedAt(t *testing.T) {
	secret := testSecret

	// Создаем токен с будущей датой выдачи
	futureIssuedAt := time.Now().Add(1 * time.Hour)
	claims := Claims{
		UserID: "123",
		Email:  "test@example.com",
		Role:   "client",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(futureIssuedAt.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(futureIssuedAt),
			NotBefore: jwt.NewNumericDate(futureIssuedAt),
			ID:        "token-id",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	futureToken, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	// Токен с будущей датой выдачи должен считаться невалидным
	claimsResult, err := ValidateJWT(futureToken, secret)
	assert.Error(t, err, "Token with future issued at should be invalid")
	assert.Nil(t, claimsResult)
}

func TestJWTStructure(t *testing.T) {
	secret := testSecret
	userID := "user-123"
	email := "user@example.com"
	role := "client"

	token, err := GenerateJWT(userID, email, role, secret, 24)
	require.NoError(t, err)

	parser := jwt.Parser{}
	parsed, _, err := parser.ParseUnverified(token, &Claims{})
	assert.NoError(t, err)

	claims, ok := parsed.Claims.(*Claims)
	assert.True(t, ok)
	assert.NotEmpty(t, claims.ID)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
}

func TestGenerateJWT_NegativeExpiration_UsesDefault(t *testing.T) {
	token, err := GenerateJWT("user-1", "a@b.com", "client", testSecret, -5)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ValidateJWT(token, testSecret)
	require.NoError(t, err)
	// Token should still be valid (default 24h applied)
	assert.NotNil(t, claims.ExpiresAt)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
}

func TestValidateJWT_InvalidTokenSignature(t *testing.T) {
	// Create a valid token with one secret, then try to validate with another
	token, err := GenerateJWT("user-1", "a@b.com", "client", "secret-a", 24)
	require.NoError(t, err)

	claims, err := ValidateJWT(token, "secret-b")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestClaimsJSONMarshaling(t *testing.T) {
	claims := Claims{
		UserID: "user-42",
		Email:  "user@test.com",
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        "test-jti",
		},
	}

	// Verify that claims can be serialized/deserialized
	data, err := json.Marshal(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded Claims
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, claims.UserID, decoded.UserID)
	assert.Equal(t, claims.Email, decoded.Email)
	assert.Equal(t, claims.Role, decoded.Role)
}
