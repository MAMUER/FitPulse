package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHelpers_SafeIntToInt32(t *testing.T) {
	tests := []struct {
		name string
		v    int
		want int32
	}{
		{"normal value", 100, 100},
		{"max int32", 2147483647, 2147483647},
		{"overflow", 2147483648, 2147483647},
		{"underflow", -2147483649, -2147483648},
		{"negative", -100, -100},
		{"zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeIntToInt32(tt.v)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHelpers_IsValidServiceURL(t *testing.T) {
	tests := []struct {
		url      string
		prefixes []string
		want     bool
	}{
		{"http://localhost:8001", []string{"http://localhost:"}, true},
		{"https://localhost:8001", []string{"http://localhost:"}, false},
		{"http://classifier:8001", []string{"http://classifier:"}, true},
		{"ftp://localhost:8001", []string{"http://localhost:"}, false},
		{"http://evil.com", []string{"http://localhost:"}, false},
		{"", []string{"http://localhost:"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := isValidServiceURL(tt.url, tt.prefixes...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHelpers_GRPCToHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string
	}{
		{"nil error", nil, 200, ""},
		{"invalid argument", status.Error(codes.InvalidArgument, "email is required"), 400, "Укажите email"},
		{"not found", status.Error(codes.NotFound, "user not found"), 404, "Не найдено"},
		{"permission denied", status.Error(codes.PermissionDenied, "forbidden"), 404, "Не найдено"},
		{"unauthenticated", status.Error(codes.Unauthenticated, "invalid credentials"), 401, "Неверные учётные данные"},
		{"resource exhausted", status.Error(codes.ResourceExhausted, "rate limit"), 429, "Превышен лимит запросов"},
		{"internal error", status.Error(codes.Internal, "internal error"), 500, "Внутренняя ошибка сервера"},
		{"unavailable", status.Error(codes.Unavailable, "service down"), 503, "Сервис временно недоступен"},
		{"unknown code", status.Error(codes.Code(999), "unknown"), 500, "unknown"},
		{"non-grpc error", errors.New("plain error"), 500, "Внутренняя ошибка сервера"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, msg := grpcToHTTPStatus(tt.err)
			assert.Equal(t, tt.wantCode, code)
			if tt.wantMsg != "" {
				assert.Equal(t, tt.wantMsg, msg)
			}
		})
	}
}

func TestHelpers_TranslateError(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"email is required", "Укажите email"},
		{"password is required", "Укажите пароль"},
		{"invalid email format", "Некорректный формат email"},
		{"user not found", "Пользователь не найден"},
		{"email already exists", "Этот email уже зарегистрирован"},
		{"invalid credentials", "Неверный email или пароль"},
		{"unknown error", "unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := translateError(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHelpers_ContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "foo", false},
		{"", "", true},
		{"abc", "abcd", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.substr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHelpers_ContainsSubstringIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "foo", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := containsSubstringIgnoreCase(tt.s, tt.substr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHelpers_PtrFunctions(t *testing.T) {
	str := "test"
	strPtr := ptrString(str)
	assert.Equal(t, "test", *strPtr)

	i32 := int32(42)
	i32Ptr := ptrInt32(i32)
	assert.Equal(t, int32(42), *i32Ptr)

	f64 := float64(3.14)
	f64Ptr := ptrFloat64(f64)
	assert.Equal(t, float64(3.14), *f64Ptr)

	f32 := float32(2.71)
	f32Ptr := ptrFloat32(f32)
	assert.Equal(t, float32(2.71), *f32Ptr)
}
