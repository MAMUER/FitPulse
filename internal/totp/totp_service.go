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

	"github.com/MAMUER/project/internal/crypto"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	Issuer           = "FitPulse"
	BackupCodesCount = 10
	BackupCodeLength = 8
)

type TOTPSetup struct {
	Secret      string
	QRCodeURL   string
	BackupCodes []string
}

type Service struct {
	encryptor *crypto.TOTPEncryptor
}

func NewService(encryptor *crypto.TOTPEncryptor) *Service {
	return &Service{encryptor: encryptor}
}

func (s *Service) GenerateTOTPSecret(userEmail string) (*TOTPSetup, error) {
	return GenerateTOTPSecret(userEmail)
}

func (s *Service) ValidateTOTPCode(passcode string, secret string) (bool, error) {
	return ValidateTOTPCode(passcode, secret)
}

func GenerateTOTPSecret(userEmail string) (*TOTPSetup, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      Issuer,
		AccountName: userEmail,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	backupCodes, err := generateBackupCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	return &TOTPSetup{
		Secret:      key.Secret(),
		QRCodeURL:   key.URL(),
		BackupCodes: backupCodes,
	}, nil
}

func ValidateTOTPCode(passcode string, secret string) (bool, error) {
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
		return false, fmt.Errorf("TOTP validation error: %w", err)
	}
	return valid, nil
}

func ValidateBackupCode(code string, hashedCodes []string) (int, error) {
	normalizedCode := normalizeBackupCode(code)
	codeHash := hashBackupCode(normalizedCode)
	for i, hashed := range hashedCodes {
		if hashed == codeHash {
			return i, nil
		}
	}
	return -1, errors.New("invalid backup code")
}

// HashBackupCodes хеширует резервные коды перед сохранением в БД
func HashBackupCodes(codes []string) []string {
	hashed := make([]string, len(codes))
	for i, code := range codes {
		hashed[i] = hashBackupCode(normalizeBackupCode(code))
	}
	return hashed
}

func (s *Service) EncryptSecret(secret string) ([]byte, error) {
	if s == nil || s.encryptor == nil {
		return nil, errors.New("TOTP encryption service not initialized")
	}
	ciphertext, err := s.encryptor.Encrypt([]byte(secret))
	return ciphertext, fmt.Errorf("encrypt secret: %w", err)
}

func (s *Service) DecryptSecret(ciphertext []byte) (string, error) {
	if s == nil || s.encryptor == nil {
		return "", errors.New("TOTP encryption service not initialized")
	}
	plaintext, err := s.encryptor.Decrypt(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt totp secret: %w", err)
	}
	return string(plaintext), nil
}

func generateBackupCodes() ([]string, error) {
	codes := make([]string, BackupCodesCount)
	for i := 0; i < BackupCodesCount; i++ {
		bytes := make([]byte, BackupCodeLength/2)
		if _, err := rand.Read(bytes); err != nil {
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
