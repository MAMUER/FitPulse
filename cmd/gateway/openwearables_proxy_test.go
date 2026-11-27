package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSingleJoiningSlash(t *testing.T) {
	assert.Equal(t, "/a/b", singleJoiningSlash("/a", "b"))
	assert.Equal(t, "/a/b", singleJoiningSlash("/a/", "/b"))
	assert.Equal(t, "/a/b", singleJoiningSlash("/a", "/b"))
	assert.Equal(t, "/a/b", singleJoiningSlash("/a/", "b"))
}

func TestOpenWearablesProxyStripsPrefix(t *testing.T) {
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	target, err := url.Parse(backend.URL)
	assert.NoError(t, err)

	handler := http.StripPrefix("/ow-api", newPrefixProxy(target))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ow-api/auth/login", nil)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/auth/login", gotPath)
}

func TestOpenWearablesFrontendProxyKeepsPrefix(t *testing.T) {
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	target, err := url.Parse(backend.URL)
	assert.NoError(t, err)

	// SvelteKit собран с base=/open-wearables/, поэтому префикс сохраняется.
	handler := newPrefixProxy(target)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/open-wearables/", nil)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/open-wearables/", gotPath)
}
