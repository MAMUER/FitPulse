package crypto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAESGCMEncryptor(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantError bool
	}{
		{
			name:      "valid base64 key",
			key:       "AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA=",
			wantError: false,
		},
		{
			name:      "valid raw 32-byte key",
			key:       "123456789012345678901234567890 2",
			wantError: false,
		},
		{
			name:      "empty key",
			key:       "",
			wantError: true,
		},
		{
			name:      "too short key",
			key:       "short",
			wantError: true,
		},
		{
			name:      "too long key",
			key:       "this_key_is_exactly_thirty_three_chars_long",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewAESGCMEncryptor(tt.key)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, enc)
			require.NotNil(t, enc.key)
		})
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := "123456789012345678901234567890 2"
	enc, err := NewAESGCMEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("hello world")
	ciphertext, err := enc.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecryptEmpty(t *testing.T) {
	key := "123456789012345678901234567890 2"
	enc, err := NewAESGCMEncryptor(key)
	require.NoError(t, err)

	ciphertext, err := enc.Encrypt(nil)
	require.NoError(t, err)

	decrypted, err := enc.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	key := "123456789012345678901234567890 2"
	enc, err := NewAESGCMEncryptor(key)
	require.NoError(t, err)

	_, err = enc.Decrypt([]byte("too short"))
	require.Error(t, err)

	_, err = enc.Decrypt([]byte("exactly twelve bytes"))
	require.Error(t, err)
}

func TestEncryptNilEncryptor(t *testing.T) {
	var enc *AESGCMEncryptor
	_, err := enc.Encrypt([]byte("test"))
	require.Error(t, err)

	_, err = enc.Decrypt([]byte("test"))
	require.Error(t, err)
}

func TestEncryptDeterminism(t *testing.T) {
	key := "123456789012345678901234567890 2"
	enc, err := NewAESGCMEncryptor(key)
	require.NoError(t, err)

	ct1, err := enc.Encrypt([]byte("same plaintext"))
	require.NoError(t, err)

	ct2, err := enc.Encrypt([]byte("same plaintext"))
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "AES-GCM with random nonce should produce different ciphertexts")
}

func TestNewAESGCMEncryptorBase64(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("abcdefghijklmnopqrstuvwxyz012345"))
	enc, err := NewAESGCMEncryptor(key)
	require.NoError(t, err)
	require.NotNil(t, enc)
}

func TestEncryptDecryptWithBase64Key(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("abcdefghijklmnopqrstuvwxyz012345"))
	enc, err := NewAESGCMEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("base64 key roundtrip")
	ct, err := enc.Encrypt(plaintext)
	require.NoError(t, err)

	pt, err := enc.Decrypt(ct)
	require.NoError(t, err)
	assert.Equal(t, plaintext, pt)
}
