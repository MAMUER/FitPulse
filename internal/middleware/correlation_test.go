package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestCorrelationIDHTTP_GeneratesNewID(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid := r.Context().Value(CorrelationIDKey)
		assert.NotNil(t, cid)
		w.WriteHeader(http.StatusOK)
	})

	handler := CorrelationIDHTTP(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.NotEmpty(t, rr.Header().Get("X-Correlation-ID"))
}

func TestCorrelationIDHTTP_UsesExistingHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid := r.Context().Value(CorrelationIDKey)
		assert.Equal(t, "existing-cid-123", cid)
		w.WriteHeader(http.StatusOK)
	})

	handler := CorrelationIDHTTP(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("X-Correlation-ID", "existing-cid-123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "existing-cid-123", rr.Header().Get("X-Correlation-ID"))
}

func TestCorrelationIDGRPC_GeneratesNewID(t *testing.T) {
	interceptor := CorrelationIDGRPC()

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		cid := GetCorrelationID(ctx)
		assert.NotEmpty(t, cid)
		assert.NotEqual(t, correlationIDUnknown, cid)
		return "ok", nil
	}

	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Method"}, handler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.True(t, handlerCalled)
}

func TestCorrelationIDGRPC_UsesExistingMetadata(t *testing.T) {
	interceptor := CorrelationIDGRPC()

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		cid := GetCorrelationID(ctx)
		assert.Equal(t, "existing-cid", cid)
		return "ok", nil
	}

	md := metadata.New(map[string]string{
		correlationIDHeader: "existing-cid",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Method"}, handler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.True(t, handlerCalled)
}

func TestCorrelationIDGRPCClient_InjectFromContext(t *testing.T) {
	interceptor := CorrelationIDGRPCClient()

	handlerCalled := false
	handler := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		handlerCalled = true
		md, ok := metadata.FromOutgoingContext(ctx)
		assert.True(t, ok)
		vals := md.Get(correlationIDHeader)
		assert.Equal(t, []string{"ctx-cid"}, vals)
		return nil
	}

	ctx := context.WithValue(context.Background(), CorrelationIDKey, "ctx-cid")
	err := interceptor(ctx, "/test.Method", nil, nil, nil, handler)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestCorrelationIDGRPCClient_SkipsUnknown(t *testing.T) {
	interceptor := CorrelationIDGRPCClient()

	handlerCalled := false
	handler := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		handlerCalled = true
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			vals := md.Get(correlationIDHeader)
			assert.Empty(t, vals)
		}
		return nil
	}

	ctx := context.WithValue(context.Background(), CorrelationIDKey, correlationIDUnknown)
	err := interceptor(ctx, "/test.Method", nil, nil, nil, handler)

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestGetCorrelationIDValue(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		var ctx context.Context
		assert.Equal(t, correlationIDUnknown, GetCorrelationID(ctx))
	})

	t.Run("background context", func(t *testing.T) {
		assert.Equal(t, correlationIDUnknown, GetCorrelationID(context.Background()))
	})

	t.Run("valid correlation id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), CorrelationIDKey, "cid-123")
		assert.Equal(t, "cid-123", GetCorrelationID(ctx))
	})

	t.Run("missing key", func(t *testing.T) {
		assert.Equal(t, correlationIDUnknown, GetCorrelationID(context.Background()))
	})

	t.Run("wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), CorrelationIDKey, 123)
		assert.Equal(t, correlationIDUnknown, GetCorrelationID(ctx))
	})
}

func TestGetUserID(t *testing.T) {
	t.Run("valid user id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserIDKey, "user-123")
		assert.Equal(t, "user-123", GetUserID(ctx))
	})

	t.Run("missing key", func(t *testing.T) {
		assert.Equal(t, "anonymous", GetUserID(context.Background()))
	})

	t.Run("wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserIDKey, 123)
		assert.Equal(t, "anonymous", GetUserID(ctx))
	})
}
