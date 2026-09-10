package apperrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	inner := errors.New("inner error")
	err := New("CODE", "some message", inner)

	assert.Equal(t, "CODE", err.Code)
	assert.Equal(t, "some message", err.Message)
	assert.Equal(t, inner, err.Err)
	assert.Equal(t, "some message: inner error", err.Error())
	assert.Equal(t, inner, err.Unwrap())
}

func TestNewWithNilError(t *testing.T) {
	err := New("CODE", "some message", nil)

	assert.Equal(t, "CODE", err.Code)
	assert.Equal(t, "some message", err.Message)
	assert.Nil(t, err.Err)
	assert.Equal(t, "some message", err.Error())
	assert.Nil(t, err.Unwrap())
}

func TestWithMessage(t *testing.T) {
	inner := errors.New("inner error")
	err := WithMessage(inner, "wrapped")

	assert.Equal(t, "INTERNAL", err.Code)
	assert.Equal(t, "wrapped", err.Message)
	assert.Equal(t, inner, err.Err)
	assert.Equal(t, "wrapped: inner error", err.Error())
}

func TestNotFound(t *testing.T) {
	err := NotFound("resource not found")

	assert.Equal(t, "NOT_FOUND", err.Code)
	assert.Equal(t, "resource not found", err.Message)
	assert.Nil(t, err.Err)
	assert.Equal(t, "resource not found", err.Error())
	assert.Nil(t, err.Unwrap())
}

func TestUnauthorized(t *testing.T) {
	err := Unauthorized("invalid token")

	assert.Equal(t, "UNAUTHORIZED", err.Code)
	assert.Equal(t, "invalid token", err.Message)
	assert.Nil(t, err.Err)
	assert.Equal(t, "invalid token", err.Error())
}

func TestForbidden(t *testing.T) {
	err := Forbidden("no access")

	assert.Equal(t, "FORBIDDEN", err.Code)
	assert.Equal(t, "no access", err.Message)
	assert.Nil(t, err.Err)
}

func TestInvalidArgument(t *testing.T) {
	err := InvalidArgument("bad input")

	assert.Equal(t, "INVALID_ARGUMENT", err.Code)
	assert.Equal(t, "bad input", err.Message)
	assert.Nil(t, err.Err)
}

func TestConflict(t *testing.T) {
	err := Conflict("already exists")

	assert.Equal(t, "CONFLICT", err.Code)
	assert.Equal(t, "already exists", err.Message)
	assert.Nil(t, err.Err)
}

func TestInternal(t *testing.T) {
	inner := errors.New("panic")
	err := Internal("something failed", inner)

	assert.Equal(t, "INTERNAL", err.Code)
	assert.Equal(t, "something failed", err.Message)
	assert.Equal(t, inner, err.Err)
	assert.Equal(t, "something failed: panic", err.Error())
}

func TestInternalWithNilError(t *testing.T) {
	err := Internal("something failed", nil)

	assert.Equal(t, "INTERNAL", err.Code)
	assert.Equal(t, "something failed", err.Message)
	assert.Nil(t, err.Err)
	assert.Equal(t, "something failed", err.Error())
}

func TestUnavailable(t *testing.T) {
	err := Unavailable("service down")

	assert.Equal(t, "UNAVAILABLE", err.Code)
	assert.Equal(t, "service down", err.Message)
	assert.Nil(t, err.Err)
}

func TestValidation(t *testing.T) {
	err := Validation("invalid input")

	assert.Equal(t, "VALIDATION", err.Code)
	assert.Equal(t, "invalid input", err.Message)
	assert.Nil(t, err.Err)
}

func TestRateLimited(t *testing.T) {
	err := RateLimited("too many requests")

	assert.Equal(t, "RATE_LIMITED", err.Code)
	assert.Equal(t, "too many requests", err.Message)
	assert.Nil(t, err.Err)
}

func TestWrap(t *testing.T) {
	t.Run("wraps error when provided", func(t *testing.T) {
		inner := errors.New("inner error")
		err := Wrap(inner, "CUSTOM", "custom message")

		assert.Equal(t, "CUSTOM", err.Code)
		assert.Equal(t, "custom message", err.Message)
		assert.Equal(t, inner, err.Err)
		assert.Equal(t, "custom message: inner error", err.Error())
	})

	t.Run("creates AppError without inner error when nil", func(t *testing.T) {
		err := Wrap(nil, "CUSTOM", "custom message")

		assert.Equal(t, "CUSTOM", err.Code)
		assert.Equal(t, "custom message", err.Message)
		assert.Nil(t, err.Err)
		assert.Equal(t, "custom message", err.Error())
	})
}

func TestCode(t *testing.T) {
	t.Run("returns code for AppError", func(t *testing.T) {
		err := NotFound("not found")
		assert.Equal(t, "NOT_FOUND", Code(err))
	})

	t.Run("returns code for wrapped AppError", func(t *testing.T) {
		inner := errors.New("some error")
		err := Wrap(inner, "CUSTOM", "message")
		assert.Equal(t, "CUSTOM", Code(err))
	})

	t.Run("returns INTERNAL for non-AppError", func(t *testing.T) {
		err := errors.New("plain error")
		assert.Equal(t, "INTERNAL", Code(err))
	})

	t.Run("returns INTERNAL for nil", func(t *testing.T) {
		assert.Equal(t, "INTERNAL", Code(nil))
	})
}

func TestMessage(t *testing.T) {
	t.Run("returns message for AppError", func(t *testing.T) {
		err := NotFound("not found")
		assert.Equal(t, "not found", Message(err))
	})

	t.Run("returns message for wrapped AppError", func(t *testing.T) {
		inner := errors.New("some error")
		err := Wrap(inner, "CUSTOM", "custom message")
		assert.Equal(t, "custom message", Message(err))
	})

	t.Run("returns default for non-AppError", func(t *testing.T) {
		err := errors.New("plain error")
		assert.Equal(t, "internal error", Message(err))
	})

	t.Run("returns default for nil", func(t *testing.T) {
		assert.Equal(t, "internal error", Message(nil))
	})
}

func TestIsAppError(t *testing.T) {
	t.Run("returns true for AppError", func(t *testing.T) {
		err := NotFound("not found")
		assert.True(t, IsAppError(err))
	})

	t.Run("returns false for plain error", func(t *testing.T) {
		err := errors.New("plain error")
		assert.False(t, IsAppError(err))
	})

	t.Run("returns false for nil", func(t *testing.T) {
		assert.False(t, IsAppError(nil))
	})
}

func TestIsNotFound(t *testing.T) {
	t.Run("returns true for NOT_FOUND", func(t *testing.T) {
		err := NotFound("not found")
		assert.True(t, IsNotFound(err))
	})

	t.Run("returns false for FORBIDDEN", func(t *testing.T) {
		err := Forbidden("no access")
		assert.False(t, IsNotFound(err))
	})

	t.Run("returns false for plain error", func(t *testing.T) {
		err := errors.New("plain error")
		assert.False(t, IsNotFound(err))
	})
}

func TestIsUnauthorized(t *testing.T) {
	t.Run("returns true for UNAUTHORIZED", func(t *testing.T) {
		err := Unauthorized("no auth")
		assert.True(t, IsUnauthorized(err))
	})

	t.Run("returns false for other codes", func(t *testing.T) {
		err := Forbidden("no access")
		assert.False(t, IsUnauthorized(err))
	})

	t.Run("returns false for plain error", func(t *testing.T) {
		err := errors.New("plain error")
		assert.False(t, IsUnauthorized(err))
	})
}

func TestIsForbidden(t *testing.T) {
	t.Run("returns true for FORBIDDEN", func(t *testing.T) {
		err := Forbidden("no access")
		assert.True(t, IsForbidden(err))
	})

	t.Run("returns false for other codes", func(t *testing.T) {
		err := Unauthorized("no auth")
		assert.False(t, IsForbidden(err))
	})
}

func TestIsValidation(t *testing.T) {
	t.Run("returns true for VALIDATION", func(t *testing.T) {
		err := Validation("bad input")
		assert.True(t, IsValidation(err))
	})

	t.Run("returns true for INVALID_ARGUMENT", func(t *testing.T) {
		err := InvalidArgument("bad input")
		assert.True(t, IsValidation(err))
	})

	t.Run("returns false for other codes", func(t *testing.T) {
		err := NotFound("not found")
		assert.False(t, IsValidation(err))
	})
}

func TestIsConflict(t *testing.T) {
	t.Run("returns true for CONFLICT", func(t *testing.T) {
		err := Conflict("already exists")
		assert.True(t, IsConflict(err))
	})

	t.Run("returns false for other codes", func(t *testing.T) {
		err := NotFound("not found")
		assert.False(t, IsConflict(err))
	})
}

func TestGRPCCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"NOT_FOUND", NotFound("x"), "NotFound"},
		{"UNAUTHORIZED", Unauthorized("x"), "Unauthenticated"},
		{"FORBIDDEN", Forbidden("x"), "PermissionDenied"},
		{"INVALID_ARGUMENT", InvalidArgument("x"), "InvalidArgument"},
		{"VALIDATION", Validation("x"), "InvalidArgument"},
		{"CONFLICT", Conflict("x"), "AlreadyExists"},
		{"UNAVAILABLE", Unavailable("x"), "Unavailable"},
		{"RATE_LIMITED", RateLimited("x"), "ResourceExhausted"},
		{"plain error", errors.New("x"), "Internal"},
		{"nil", nil, "Internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GRPCCode(tt.err))
		})
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"NOT_FOUND", NotFound("x"), 404},
		{"UNAUTHORIZED", Unauthorized("x"), 401},
		{"FORBIDDEN", Forbidden("x"), 403},
		{"INVALID_ARGUMENT", InvalidArgument("x"), 400},
		{"VALIDATION", Validation("x"), 400},
		{"CONFLICT", Conflict("x"), 409},
		{"UNAVAILABLE", Unavailable("x"), 503},
		{"RATE_LIMITED", RateLimited("x"), 429},
		{"plain error", errors.New("x"), 500},
		{"nil", nil, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, HTTPStatus(tt.err))
		})
	}
}

func TestToGRPCStatus(t *testing.T) {
	err := NotFound("x")
	assert.Equal(t, GRPCCode(err), ToGRPCStatus(err))
}

func TestFormat(t *testing.T) {
	t.Run("formats AppError with inner error", func(t *testing.T) {
		inner := errors.New("inner error")
		err := New("CODE", "message", inner)
		assert.Equal(t, "message: inner error", Format(err))
	})

	t.Run("formats AppError without inner error", func(t *testing.T) {
		err := NotFound("not found")
		assert.Equal(t, "not found", Format(err))
	})

	t.Run("formats plain error", func(t *testing.T) {
		err := errors.New("plain error")
		assert.Equal(t, "plain error", Format(err))
	})
}

func TestErrorImplementsErrorInterface(t *testing.T) {
	err := NotFound("test")
	var e error = err
	assert.NotNil(t, e)
	assert.Equal(t, "test", e.Error())
}

func TestErrorsAs(t *testing.T) {
	t.Run("can extract AppError from wrapped error", func(t *testing.T) {
		inner := errors.New("inner")
		err := Wrap(inner, "CODE", "message")

		var appErr *AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, "CODE", appErr.Code)
		assert.Equal(t, "message", appErr.Message)
	})

	t.Run("returns false for non-AppError", func(t *testing.T) {
		plain := errors.New("plain")
		var appErr *AppError
		assert.False(t, errors.As(plain, &appErr))
	})
}
