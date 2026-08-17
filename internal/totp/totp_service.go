// Package totp provides TOTP secret generation, validation, and backup-code helpers.
package totp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/MAMUER/project/internal/crypto"
)

const (
	Issuer           = "FitPulse"
	BackupCodesCount = 10
	BackupCodeLength = 8
)

// TOTPSetup contains the data needed to complete TOTP enrollment.
type TOTPSetup struct {
	Secret      string
	QRCodeURL   string
	BackupCodes []string
}

// Service manages TOTP lifecycle: secret generation, passcode validation,
// backup-code verification, and secret encryption/decryption.
type Service struct {
	encryptor *crypto.AESGCMEncryptor
}

// NewService creates a TOTP service backed by the provided encryptor.
// If encryptor is nil, encryption/decryption methods will return errors.
func NewService(encryptor *crypto.AESGCMEncryptor) *Service {
	return &Service{encryptor: encryptor}
}

// GenerateTOTPSecret creates a new TOTP key and backup codes for the given user.
func (s *Service) GenerateTOTPSecret(userEmail string) (*TOTPSetup, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      Issuer,
		AccountName: userEmail,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("generate TOTP key: %w", err)
	}

	backupCodes, err := generateBackupCodes()
	if err != nil {
		return nil, fmt.Errorf("generate backup codes: %w", err)
	}

	return &TOTPSetup{
		Secret:      key.Secret(),
		QRCodeURL:   key.URL(),
		BackupCodes: backupCodes,
	}, nil
}

// ValidateTOTPCode checks a 6-digit passcode against the TOTP secret.
// It allows a skew of 1 period (30 seconds) to account for clock drift.
func (s *Service) ValidateTOTPCode(passcode, secret string) (bool, error) {
	if len(passcode) != 6 {
		return false, nil
	}

	valid, err := totp.ValidateCustom(passcode, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return false, fmt.Errorf("validate TOTP code: %w", err)
	}
	return valid, nil
}

// ValidateBackupCode checks a backup code against a list of hashed backup codes.
// It returns the index of the matched code, or -1 with ErrInvalidBackupCode if not found.
func ValidateBackupCode(code string, hashedCodes []string) (int, error) {
	normalizedCode := normalizeBackupCode(code)
	codeHash := hashBackupCode(normalizedCode)
	for i, hashed := range hashedCodes {
		if hashed == codeHash {
			return i, nil
		}
	}
	return -1, ErrInvalidBackupCode
}

// ErrInvalidBackupCode is returned when a backup code does not match any stored hash.
var ErrInvalidBackupCode = errors.New("invalid backup code")

// HashBackupCodes hashes backup codes before storing them in the database.
func HashBackupCodes(codes []string) []string {
	hashed := make([]string, len(codes))
	for i, code := range codes {
		hashed[i] = hashBackupCode(normalizeBackupCode(code))
	}
	return hashed
}

// EncryptSecret encrypts a TOTP secret using the service encryptor.
func (s *Service) EncryptSecret(secret string) ([]byte, error) {
	if s == nil || s.encryptor == nil {
		return nil, errors.New("TOTP encryption service not initialized")
	}
	ciphertext, err := s.encryptor.Encrypt([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("encrypt TOTP secret: %w", err)
	}
	return ciphertext, nil
}

// DecryptSecret decrypts a TOTP secret using the service encryptor.
func (s *Service) DecryptSecret(ciphertext []byte) (string, error) {
	if s == nil || s.encryptor == nil {
		return "", errors.New("TOTP encryption service not initialized")
	}
	plaintext, err := s.encryptor.Decrypt(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt TOTP secret: %w", err)
	}
	return string(plaintext), nil
}

var backupCodeRandomReader = rand.Reader

func generateBackupCodes() ([]string, error) {
	codes := make([]string, BackupCodesCount)
	for i := 0; i < BackupCodesCount; i++ {
		bytes := make([]byte, BackupCodeLength/2)
		if _, err := backupCodeRandomReader.Read(bytes); err != nil {
			return nil, fmt.Errorf("generate backup codes: %w", err)
		}
		raw := hex.EncodeToString(bytes)
		codes[i] = fmt.Sprintf("%s-%s", raw[:4], raw[4:])
	}
	return codes, nil
}

func hashBackupCode(code string) string {
	hash := sha256.Sum256([]byte(code))
	return hex.EncodeToString(hash[:])
}

func normalizeBackupCode(code string) string {
	code = strings.TrimSpace(code)
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	return strings.ToLower(code)
}
