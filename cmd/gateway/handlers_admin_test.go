package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	"github.com/MAMUER/project/internal/middleware"
)

func TestAdmin_AdminListUsersHandler_Unauthorized(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/admin/users", nil)

	g.adminListUsersHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdmin_AdminListUsersHandler_Success(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "admin-123"), "GET", "/api/v1/admin/users", nil)

	g.adminListUsersHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "status")
}

func TestAdmin_AdminListUsersHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &errorUserServiceClient{err: grpcError(codes.PermissionDenied, "forbidden")}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.WithValue(context.Background(), middleware.UserIDKey, "admin-123"), "GET", "/api/v1/admin/users", nil)

	g.adminListUsersHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdmin_AdminListInvitesHandler_Success(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/admin/invites?page=1&page_size=10", nil)

	g.adminListInvitesHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "status")
}

func TestAdmin_AdminListInvitesHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &errorUserServiceClient{err: grpcError(codes.Internal, "internal error")}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/admin/invites?page=1&page_size=10", nil)

	g.adminListInvitesHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAdmin_AdminCreateInviteHandler_InvalidJSON(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/admin/invites", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	g.adminCreateInviteHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdmin_AdminCreateInviteHandler_Success(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	reqBody := []byte(`{"role":"client","specialty":"general","max_uses":10}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/admin/invites", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.adminCreateInviteHandler(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "status")
}

func TestAdmin_AdminRevokeInviteHandler_MissingCode(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/api/v1/admin/invites", nil)

	g.adminRevokeInviteHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdmin_AdminRevokeInviteHandler_Success(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/api/v1/admin/invites?code=INV-123", nil)

	g.adminRevokeInviteHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "status")
}

func TestAdmin_AdminRevokeInviteHandler_GRPCError(t *testing.T) {
	g := newTestGateway()
	g.userClient = &errorUserServiceClient{err: grpcError(codes.NotFound, "invite not found")}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/api/v1/admin/invites?code=INV-123", nil)

	g.adminRevokeInviteHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdmin_ParsePagination(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantPage int
		wantSize int
	}{
		{"default", "", 1, 20},
		{"custom page", "page=2&page_size=50", 2, 50},
		{"invalid page", "page=abc&page_size=10", 1, 10},
		{"zero page", "page=0&page_size=10", 1, 10},
		{"negative page", "page=-1&page_size=10", 1, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/?"+tt.query, nil)
			page, pageSize := parsePagination(req)
			assert.Equal(t, tt.wantPage, page)
			assert.Equal(t, tt.wantSize, pageSize)
		})
	}
}
