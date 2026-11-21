package main

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProtectedRoutesAreMountedAtAPIV1 — регрессионный тест на двойной префикс.
//
// registerProtectedRoutes регистрирует защищённые маршруты внутри группы. Если группу
// задать через r.Route("/api/v1", ...), chi смонтирует subrouter по /api/v1, а
// пути внутри ("/api/v1/profile") сложатся в /api/v1/api/v1/profile — и все
// защищённые эндпоинты начнут отдавать 404. Именно это поймал E2E-тест
// "Test Deployment (E2E API)": 12 из 26 проверок падали с 404.
func TestProtectedRoutesAreMountedAtAPIV1(t *testing.T) {
	g := newTestGateway()
	router := g.registerRoutes()

	registered := make(map[string]bool)
	walkErr := chi.Walk(router, func(_, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered[route] = true
		return nil
	})
	require.NoError(t, walkErr)

	for _, path := range []string{
		"/api/v1/profile",
		"/api/v1/biometrics",
		"/api/v1/logout",
		"/api/v1/training/plans",
		"/api/v1/training/progress",
		"/api/v1/chat",
		"/api/v1/auth/2fa/setup",
	} {
		assert.Truef(t, registered[path], "маршрут %s не зарегистрирован (проверьте двойной префикс /api/v1)", path)
	}

	for route := range registered {
		assert.NotContainsf(t, route, "/api/v1/api/v1", "двойной префикс в маршруте %s", route)
	}
}
