// Package webhook provides Open Wearables webhook handling.
package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
)

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

	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write(body); err != nil {
		return fmt.Errorf("failed to compute HMAC: %w", err)
	}
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return ErrInvalidSignature
	}

	r.Body = io.NopCloser(bytes.NewReader(body))

	return nil
}
