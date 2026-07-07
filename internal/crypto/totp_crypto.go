// Package crypto provides AES-GCM encryption helpers.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type AESGCMEncryptor struct {
	key []byte
}

func InitAESGCMEncryption() error {
	_, err := NewAESGCMEncryptor(os.Getenv("DEVICE_TOKEN_ENCRYPTION_KEY"))
	return err
}

func NewAESGCMEncryptor(keyMaterial string) (*AESGCMEncryptor, error) {
	keyMaterial = strings.TrimSpace(keyMaterial)
	if keyMaterial == "" {
		return nil, errors.New("encryption key environment variable is required")
	}

	key, err := base64.StdEncoding.DecodeString(keyMaterial)
	if err != nil {
		key = []byte(keyMaterial)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (256 bits), got %d", len(key))
	}

	return &AESGCMEncryptor{key: key}, nil
}

func (e *AESGCMEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	if e == nil || e.key == nil {
		return nil, errors.New("encryption not initialized")
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("read nonce: %w", err)
	}

	return aesGCM.Seal(nonce, nonce, plaintext, nil), nil
}

func (e *AESGCMEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if e == nil || e.key == nil {
		return nil, errors.New("encryption not initialized")
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt data: %w", err)
	}

	return plaintext, nil
}

type TOTPEncryptor struct {
	key []byte
}

func InitTOTPEncryption() error {
	_, err := NewTOTPEncryptor(os.Getenv("TOTP_ENCRYPTION_KEY"))
	return err
}

func NewTOTPEncryptor(keyMaterial string) (*TOTPEncryptor, error) {
	enc, err := NewAESGCMEncryptor(keyMaterial)
	if err != nil {
		return nil, err
	}
	return &TOTPEncryptor{key: enc.key}, nil
}

func (e *TOTPEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	enc := &AESGCMEncryptor{key: e.key}
	return enc.Encrypt(plaintext)
}

func (e *TOTPEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	enc := &AESGCMEncryptor{key: e.key}
	return enc.Decrypt(ciphertext)
}
