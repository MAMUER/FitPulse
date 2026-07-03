// Package auth provides authentication and authorization utilities.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// SignResponse вычисляет HMAC-SHA256 подпись байтов ответа
// Принимает []byte чтобы гарантировать подпись тех же байтов, что отправляются клиенту
func SignResponse(data []byte, secret string) (string, error) {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

// SignResponseObject вычисляет подпись JSON-объекта (для обратной совместимости)
func SignResponseObject(data interface{}, secret string) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal object: %w", err)
	}
	return SignResponse(jsonData, secret)
}

// VerifyResponse проверяет подпись ответа
func VerifyResponse(data []byte, signature, secret string) bool {
	expected, err := SignResponse(data, secret)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(expected), []byte(signature))
}
