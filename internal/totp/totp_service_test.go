package totp

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp"
	otpLib "github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/crypto"
)

func TestGenerateTOTPSecret(t *testing.T) {
	svc := NewService(nil)

	setup, err := svc.GenerateTOTPSecret("user@example.com")
	require.NoError(t, err)
	require.NotNil(t, setup)
	require.NotEmpty(t, setup.Secret)
	require.NotEmpty(t, setup.QRCodeURL)
	require.Len(t, setup.BackupCodes, BackupCodesCount)
}

func TestGenerateTOTPSecret_EmptyEmail(t *testing.T) {
	svc := NewService(nil)

	setup, err := svc.GenerateTOTPSecret("")
	require.Error(t, err)
	assert.Nil(t, setup)
}

func TestValidateTOTPCode_Valid(t *testing.T) {
	svc := NewService(nil)

	setup, err := svc.GenerateTOTPSecret("user@example.com")
	require.NoError(t, err)

	passcode, err := otpLib.GenerateCodeCustom(setup.Secret, time.Now().UTC(), otpLib.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	require.NoError(t, err)

	valid, err := svc.ValidateTOTPCode(passcode, setup.Secret)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestValidateTOTPCode_Invalid(t *testing.T) {
	svc := NewService(nil)

	setup, err := svc.GenerateTOTPSecret("user@example.com")
	require.NoError(t, err)

	valid, err := svc.ValidateTOTPCode("000000", setup.Secret)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestValidateTOTPCode_WrongLength(t *testing.T) {
	svc := NewService(nil)

	setup, err := svc.GenerateTOTPSecret("user@example.com")
	require.NoError(t, err)

	valid, err := svc.ValidateTOTPCode("12345", setup.Secret)
	require.NoError(t, err)
	assert.False(t, valid)

	valid, err = svc.ValidateTOTPCode("1234567", setup.Secret)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestValidateBackupCode(t *testing.T) {
	svc := NewService(nil)
	setup, err := svc.GenerateTOTPSecret("user@example.com")
	require.NoError(t, err)

	hashedCodes := HashBackupCodes(setup.BackupCodes)

	idx, err := ValidateBackupCode(setup.BackupCodes[0], hashedCodes)
	require.NoError(t, err)
	assert.Equal(t, 0, idx)

	idx, err = ValidateBackupCode(strings.ToUpper(setup.BackupCodes[0]), hashedCodes)
	require.NoError(t, err)
	assert.Equal(t, 0, idx)

	idx, err = ValidateBackupCode(setup.BackupCodes[0], hashedCodes)
	require.NoError(t, err)
	assert.Equal(t, 0, idx)

	_, err = ValidateBackupCode("invalid-code", hashedCodes)
	assert.ErrorIs(t, err, ErrInvalidBackupCode)
}

func TestHashBackupCodes(t *testing.T) {
	codes := []string{"abcd-1234", "efgh-5678"}
	hashed := HashBackupCodes(codes)

	require.Len(t, hashed, 2)
	assert.NotEqual(t, codes[0], hashed[0])
	assert.NotEqual(t, codes[1], hashed[1])
	assert.NotContains(t, hashed[0], "-")
	assert.NotContains(t, hashed[1], "-")
}

func TestEncryptDecryptSecret(t *testing.T) {
	encryptor, err := crypto.NewAESGCMEncryptor("1234567890123456789012345678901@") // 32 bytes, not valid base64
	require.NoError(t, err)

	svc := NewService(encryptor)
	originalSecret := "JBSWY3DPEHPK3PXP"

	ciphertext, err := svc.EncryptSecret(originalSecret)
	require.NoError(t, err)
	assert.NotEqual(t, originalSecret, string(ciphertext))

	decrypted, err := svc.DecryptSecret(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, originalSecret, decrypted)
}

func TestEncryptSecret_EncryptorError(t *testing.T) {
	oldReader := rand.Reader
	defer func() { rand.Reader = oldReader }()
	rand.Reader = &errorReader{}

	encryptor, err := crypto.NewAESGCMEncryptor("1234567890123456789012345678901@")
	require.NoError(t, err)

	svc := NewService(encryptor)
	_, err = svc.EncryptSecret("secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encrypt TOTP secret")
}

func TestDecryptSecret_DecryptorError(t *testing.T) {
	key1 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32)))
	key2 := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32)))

	encryptor, err := crypto.NewAESGCMEncryptor(key1)
	require.NoError(t, err)

	otherEncryptor, err := crypto.NewAESGCMEncryptor(key2)
	require.NoError(t, err)

	ciphertext, err := otherEncryptor.Encrypt([]byte("secret"))
	require.NoError(t, err)

	svc := NewService(encryptor)
	_, err = svc.DecryptSecret(ciphertext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decrypt TOTP secret")
}

func TestEncryptSecret_NilService(t *testing.T) {
	var svc *Service

	_, err := svc.EncryptSecret("secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestDecryptSecret_NilService(t *testing.T) {
	var svc *Service

	_, err := svc.DecryptSecret([]byte("ciphertext"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestEncryptSecret_NilEncryptor(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.EncryptSecret("secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestDecryptSecret_NilEncryptor(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.DecryptSecret([]byte("ciphertext"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestBackupCodeNormalization(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "abcd-1234", "abcd1234"},
		{"uppercase", "ABCD-1234", "abcd1234"},
		{"mixed case", "AbCd-1234", "abcd1234"},
		{"no dash", "abcd1234", "abcd1234"},
		{"spaces instead of dash", "abcd 1234", "abcd1234"},
		{"with spaces", "  abcd-1234  ", "abcd1234"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeBackupCode(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBackupCodeLength(t *testing.T) {
	svc := NewService(nil)
	setup, err := svc.GenerateTOTPSecret("user@example.com")
	require.NoError(t, err)

	for _, code := range setup.BackupCodes {
		assert.Len(t, code, BackupCodeLength+1) // XXXX-XXXX = 9 chars
		assert.Contains(t, code, "-")
	}
}

func TestQRCodeURLContainsIssuer(t *testing.T) {
	svc := NewService(nil)
	setup, err := svc.GenerateTOTPSecret("user@example.com")
	require.NoError(t, err)

	assert.Contains(t, setup.QRCodeURL, Issuer)
	assert.Contains(t, setup.QRCodeURL, "user@example.com")
}

func TestSecretFormat(t *testing.T) {
	svc := NewService(nil)
	setup, err := svc.GenerateTOTPSecret("user@example.com")
	require.NoError(t, err)

	assert.Len(t, setup.Secret, 32) // Base32 encoded 20-byte secret
	assert.Regexp(t, `^[A-Z2-7]+$`, setup.Secret)
}

func TestValidateTOTPCode_InvalidSecret(t *testing.T) {
	svc := NewService(nil)

	valid, err := svc.ValidateTOTPCode("123456", "invalid-secret-format")
	assert.Error(t, err)
	assert.False(t, valid)
}

func TestGenerateTOTPSecret_BackupCodeGenerationFailure(t *testing.T) {
	oldReader := backupCodeRandomReader
	defer func() { backupCodeRandomReader = oldReader }()
	backupCodeRandomReader = &errorReader{}

	svc := NewService(nil)
	setup, err := svc.GenerateTOTPSecret("user@example.com")
	assert.Error(t, err)
	assert.Nil(t, setup)
	assert.Contains(t, err.Error(), "generate backup codes")
}

func TestGenerateBackupCodes_Failure(t *testing.T) {
	oldReader := backupCodeRandomReader
	defer func() { backupCodeRandomReader = oldReader }()
	backupCodeRandomReader = &errorReader{}

	codes, err := generateBackupCodes()
	assert.Error(t, err)
	assert.Nil(t, codes)
}

type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}
