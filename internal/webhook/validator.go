// Package webhook provides Open Wearables webhook handling.
package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
)

// newMac is a package-level hook so tests can force mac.Write to fail.
var newMac = func(secret []byte) hash.Hash {
	return hmac.New(sha256.New, secret)
}

// ValidateSignature validates HMAC-SHA256 signature from Open Wearables
// Signature is expected in X-Open-Wearables-Signature header
func ValidateSignature(secret []byte, r *http.Request) error {
	signature := r.Header.Get("X-Open-Wearables-Signature")
	if signature == "" {
		return ErrInvalidSignature
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	if len(body) == 0 {
		return errors.New("empty request body")
	}

	mac := newMac(secret)
	if _, err := mac.Write(body); err != nil {
		return fmt.Errorf("failed to hash request body: %w", err)
	}
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return ErrInvalidSignature
	}

	r.Body = io.NopCloser(bytes.NewReader(body))

	return nil
}
