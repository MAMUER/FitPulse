package db

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmailHash(t *testing.T) {
	hash1 := EmailHash("Test@Example.com")
	hash3 := EmailHash("other@example.com")

	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 32)
	assert.Equal(t, strings.ToLower(hash1), hash1)
	assert.NotEqual(t, hash1, hash3)
	assert.Equal(t, hash1, EmailHash("Test@Example.com"))
}

func TestEmailHash_Empty(t *testing.T) {
	hash := EmailHash("")
	assert.Empty(t, hash)
	assert.Len(t, hash, 0)
}
