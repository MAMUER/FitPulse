package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/MAMUER/project/internal/logger"
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
	page, pageSize := parsePagination(httptest.NewRequest(http.MethodGet, "/", nil))
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
	g := &gateway{log: &logger.Logger{Logger: zap.NewNop()}}
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

func TestMapGRPCToHTTP(t *testing.T) {
	// codes.OK -> 200, empty msg
	st := status.New(codes.OK, "")
	code, msg := mapGRPCToHTTP(st)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "", msg)

	// codes.NotFound -> 404, fixed msg
	st = status.New(codes.NotFound, "something")
	code, msg = mapGRPCToHTTP(st)
	assert.Equal(t, http.StatusNotFound, code)
	assert.Equal(t, "Не найдено", msg)

	// codes.InvalidArgument -> 400, translated msg
	st = status.New(codes.InvalidArgument, "email is required")
	code, msg = mapGRPCToHTTP(st)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "Укажите email", msg)

	// codes.Unauthenticated -> 401
	st = status.New(codes.Unauthenticated, "bad token")
	code, msg = mapGRPCToHTTP(st)
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Equal(t, "Неверные учётные данные", msg)

	// codes.Internal -> 500
	st = status.New(codes.Internal, "boom")
	code, msg = mapGRPCToHTTP(st)
	assert.Equal(t, http.StatusInternalServerError, code)
	assert.Equal(t, "Внутренняя ошибка сервера", msg)

	// Unknown grpc code -> 500, translated error msg
	st = status.New(codes.Code(999), "custom grpc error")
	code, msg = mapGRPCToHTTP(st)
	assert.Equal(t, http.StatusInternalServerError, code)
	assert.Equal(t, "custom grpc error", msg)
}

func TestGrpcToHTTPStatus_Codes(t *testing.T) {
	// codes.ResourceExhausted -> 429
	code, msg := grpcToHTTPStatus(grpcError(codes.ResourceExhausted, "too many"))
	assert.Equal(t, http.StatusTooManyRequests, code)
	assert.Equal(t, "Превышен лимит запросов", msg)

	// codes.AlreadyExists -> 409, translated
	code, msg = grpcToHTTPStatus(grpcError(codes.AlreadyExists, "email already exists"))
	assert.Equal(t, http.StatusConflict, code)
	assert.Equal(t, "Этот email уже зарегистрирован", msg)

	// codes.PermissionDenied -> 404
	code, msg = grpcToHTTPStatus(grpcError(codes.PermissionDenied, "no access"))
	assert.Equal(t, http.StatusNotFound, code)
	assert.Equal(t, "Не найдено", msg)

	// codes.DeadlineExceeded -> 504
	code, msg = grpcToHTTPStatus(grpcError(codes.DeadlineExceeded, "timeout"))
	assert.Equal(t, http.StatusGatewayTimeout, code)
	assert.Equal(t, "Превышено время ожидания", msg)
}

func TestTranslateError_More(t *testing.T) {
	assert.Equal(t, "Укажите имя", translateError("full name is required"))
	assert.Equal(t, "Недопустимая роль", translateError("invalid role"))
	assert.Equal(t, "Некорректный формат email", translateError("invalid email format"))
	assert.Equal(t, "Возраст должен быть от 0 до 150", translateError("age must be between 0 and 150"))
	assert.Equal(t, "Пользователь не найден", translateError("user not found"))
	assert.Equal(t, "Этот email уже зарегистрирован", translateError("email already exists"))
	assert.Equal(t, "Неверный email или пароль", translateError("invalid credentials"))
	assert.Equal(t, "Пульс вне допустимого диапазона (30–220)", translateError("heart_rate out of valid range"))
	assert.Equal(t, "SpO₂ вне допустимого диапазона (70–100)", translateError("spo2 out of valid range"))
	assert.Equal(t, "msg passthrough", translateError("msg passthrough"))
}

func TestContainsSubstringIgnoreCase(t *testing.T) {
	assert.True(t, containsSubstringIgnoreCase("Hello World", "world"))
	assert.True(t, containsSubstringIgnoreCase("HELLO", "he"))
	assert.False(t, containsSubstringIgnoreCase("Hello", "xyz"))
	assert.True(t, containsSubstringIgnoreCase("ABC", "abc"))
	assert.False(t, containsSubstringIgnoreCase("", "a"))
}

func TestWriteJSONSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	g := &gateway{log: &logger.Logger{Logger: zap.NewNop()}}
	g.writeJSON(w, http.StatusCreated, map[string]string{"id": "abc"})
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"abc"`)
}
