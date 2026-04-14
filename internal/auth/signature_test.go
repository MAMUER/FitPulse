package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignResponse(t *testing.T) {
	secret := testSecret
	data := []byte(`{"key":"value"}`)
	sig, err := SignResponse(data, secret)
	if err != nil {
		t.Fatalf("SignResponse failed: %v", err)
	}
	if sig == "" {
		t.Error("Empty signature")
	}
}

func TestVerifyResponse(t *testing.T) {
	secret := testSecret
	data := []byte(`{"key":"value"}`)
	sig, _ := SignResponse(data, secret)

	if !VerifyResponse(data, sig, secret) {
		t.Error("VerifyResponse failed for valid signature")
	}
	if VerifyResponse(data, "invalid", secret) {
		t.Error("VerifyResponse passed for invalid signature")
	}
}

func TestSignResponse_DeterministicOutput(t *testing.T) {
	secret := testSecret
	data := []byte(`{"status":"ok"}`)

	sig1, err := SignResponse(data, secret)
	require.NoError(t, err)
	sig2, err := SignResponse(data, secret)
	require.NoError(t, err)

	assert.Equal(t, sig1, sig2, "Same input should produce same signature")
}

func TestSignResponse_DifferentDataDifferentSignature(t *testing.T) {
	secret := testSecret
	data1 := []byte(`{"key":"value1"}`)
	data2 := []byte(`{"key":"value2"}`)

	sig1, err := SignResponse(data1, secret)
	require.NoError(t, err)
	sig2, err := SignResponse(data2, secret)
	require.NoError(t, err)

	assert.NotEqual(t, sig1, sig2, "Different data should produce different signatures")
}

func TestSignResponse_DifferentSecretDifferentSignature(t *testing.T) {
	data := []byte(`{"key":"value"}`)

	sig1, err := SignResponse(data, "secret-a")
	require.NoError(t, err)
	sig2, err := SignResponse(data, "secret-b")
	require.NoError(t, err)

	assert.NotEqual(t, sig1, sig2, "Different secrets should produce different signatures")
}

func TestSignResponse_ComplexData(t *testing.T) {
	secret := testSecret
	data := []byte(`{"user_id":123,"email":"test@example.com","roles":["admin","user"],"active":true}`)

	sig, err := SignResponse(data, secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, sig)
	assert.True(t, VerifyResponse(data, sig, secret))
}

func TestSignResponse_EmptySecret(t *testing.T) {
	data := []byte(`{"key":"value"}`)

	sig, err := SignResponse(data, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, sig)

	// Empty secret is valid for HMAC (it just uses an empty key)
	assert.True(t, VerifyResponse(data, sig, ""))
}

func TestSignResponse_NilData(t *testing.T) {
	sig, err := SignResponse(nil, testSecret)
	assert.NoError(t, err)
	assert.NotEmpty(t, sig)

	// Verify with nil data
	assert.True(t, VerifyResponse(nil, sig, testSecret))
}

func TestSignResponse_EmptyData(t *testing.T) {
	sig, err := SignResponse([]byte{}, testSecret)
	assert.NoError(t, err)
	assert.NotEmpty(t, sig)

	assert.True(t, VerifyResponse([]byte{}, sig, testSecret))
}

func TestVerifyResponse_WrongSecret(t *testing.T) {
	data := []byte(`{"key":"value"}`)
	sig, err := SignResponse(data, "correct-secret")
	require.NoError(t, err)

	assert.False(t, VerifyResponse(data, sig, "wrong-secret"))
}

func TestVerifyResponse_EmptySignature(t *testing.T) {
	data := []byte(`{"key":"value"}`)
	assert.False(t, VerifyResponse(data, "", testSecret))
}

func TestVerifyResponse_MalformedSignature(t *testing.T) {
	data := []byte(`{"key":"value"}`)
	validSig, err := SignResponse(data, testSecret)
	require.NoError(t, err)

	// Truncate the signature
	malformed := validSig[:5]
	assert.False(t, VerifyResponse(data, malformed, testSecret))
}

func TestSignResponseObject_BackwardCompatibility(t *testing.T) {
	secret := testSecret
	data := map[string]string{"key": "value"}

	sig, err := SignResponseObject(data, secret)
	require.NoError(t, err)
	assert.NotEmpty(t, sig)
}

func TestSignResponse_ExactBytesMatch(t *testing.T) {
	secret := testSecret
	// Ключевой тест: подпись JSON, сериализованного двумя способами, должна совпадать
	// только если байты идентичны
	data1 := []byte(`{"status":"ok","token":"abc"}`)
	data2 := []byte(`{"status":"ok","token":"abc"}`)

	sig1, _ := SignResponse(data1, secret)
	sig2, _ := SignResponse(data2, secret)

	assert.Equal(t, sig1, sig2, "Identical bytes must produce identical signatures")

	// Но если хотя бы один байт отличается — подпись другая
	data3 := []byte(`{"status":"ok","token":"abd"}`)
	sig3, _ := SignResponse(data3, secret)
	assert.NotEqual(t, sig1, sig3, "Different byte must produce different signature")
}
