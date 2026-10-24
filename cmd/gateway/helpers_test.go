package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPtrFunctions(t *testing.T) {
	v32 := int32(42)
	assert.Equal(t, &v32, ptrInt32(42))

	vs := "hello"
	assert.Equal(t, &vs, ptrString("hello"))

	vf64 := float64(3.14)
	assert.Equal(t, &vf64, ptrFloat64(3.14))

	vf32 := float32(2.71)
	assert.Equal(t, &vf32, ptrFloat32(2.71))
}

func TestSafeIntToInt32(t *testing.T) {
	assert.Equal(t, int32(100), safeIntToInt32(100))
	assert.Equal(t, int32(2147483647), safeIntToInt32(9999999999))
	assert.Equal(t, int32(-2147483648), safeIntToInt32(-9999999999))
	assert.Equal(t, int32(0), safeIntToInt32(0))
}

func TestParsePagination(t *testing.T) {
	// Defaults
	page, pageSize := parsePagination(&http.Request{})
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, pageSize)

	// Custom values
	r := httptest.NewRequest(http.MethodGet, "/?page=3&page_size=50", nil)
	page, pageSize = parsePagination(r)
	assert.Equal(t, 3, page)
	assert.Equal(t, 50, pageSize)

	// Invalid values ignored
	r = httptest.NewRequest(http.MethodGet, "/?page=abc&page_size=xyz", nil)
	page, pageSize = parsePagination(r)
	assert.Equal(t, 1, page)
	assert.Equal(t, 20, pageSize)

	// Page size capped at 100
	r = httptest.NewRequest(http.MethodGet, "/?page_size=200", nil)
	page, pageSize = parsePagination(r)
	assert.Equal(t, 1, page)
	assert.Equal(t, 100, pageSize)
}

func TestIsValidServiceURL(t *testing.T) {
	assert.False(t, isValidServiceURL("ftp://example.com"))
	assert.False(t, isValidServiceURL("example.com"))
	assert.True(t, isValidServiceURL("http://example.com", "http://example.com"))
	assert.True(t, isValidServiceURL("https://api.service.local", "https://api.service.local"))
	assert.False(t, isValidServiceURL("https://other.service.local", "https://api.service.local"))
	assert.True(t, isValidServiceURL("https://api.service.local/v1", "https://api.service.local"))
}

func TestGrpcToHTTPStatus(t *testing.T) {
	// nil error
	code, msg := grpcToHTTPStatus(nil)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "", msg)

	// Unknown error
	code, msgUnknown := grpcToHTTPStatus(assert.AnError)
	assert.Equal(t, http.StatusInternalServerError, code)
	_ = msgUnknown
}

func TestTranslateError(t *testing.T) {
	assert.Equal(t, "Укажите email", translateError("email is required"))
	assert.Equal(t, "Укажите пароль", translateError("password is required"))
	assert.Equal(t, "unknown error msg", translateError("unknown error msg"))
	assert.Equal(t, "Пароль должен быть не менее 8 символов", translateError("password must be at least 8 chars"))
}

func TestContainsIgnoreCase(t *testing.T) {
	assert.True(t, containsIgnoreCase("Hello World", "hello"))
	assert.True(t, containsIgnoreCase("HELLO", "HELLO"))
	assert.False(t, containsIgnoreCase("Hello", "world"))
	assert.True(t, containsIgnoreCase("Test", "Test"))
	assert.False(t, containsIgnoreCase("", "a"))
}

func TestWriteJSONEncodeError(t *testing.T) {
	t.Parallel()
	// Test writeJSON with a writer that fails
	w := &failingResponseWriter{}
	g := &gateway{}
	g.writeJSON(w, http.StatusOK, map[string]string{"key": "value"})
	// Should not panic, just log error
}

type failingResponseWriter struct {
	http.ResponseWriter
}

func (f *failingResponseWriter) Header() http.Header {
	return http.Header{}
}

func (f *failingResponseWriter) WriteHeader(statusCode int) {}

func (f *failingResponseWriter) Write([]byte) (int, error) {
	return 0, assert.AnError
}
