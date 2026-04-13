package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestErrorCodeValues(t *testing.T) {
	assert.Equal(t, ErrorCode("INVALID_INPUT"), ErrInvalidInput)
	assert.Equal(t, ErrorCode("NOT_FOUND"), ErrNotFound)
	assert.Equal(t, ErrorCode("DATABASE_ERROR"), ErrDatabase)
	assert.Equal(t, ErrorCode("VALIDATION_FAILED"), ErrValidation)
	assert.Equal(t, ErrorCode("CONCURRENT_MODIFICATION"), ErrConcurrentMod)
}

func TestErrorStringWithoutCause(t *testing.T) {
	err := &Error{
		Code:    ErrInvalidInput,
		Message: "bad input",
	}
	assert.Equal(t, "INVALID_INPUT: bad input", err.Error())
}

func TestErrorStringWithCause(t *testing.T) {
	cause := assert.AnError
	err := &Error{
		Code:    ErrDatabase,
		Message: "query failed",
		Cause:   cause,
	}
	got := err.Error()
	assert.Contains(t, got, "DATABASE_ERROR")
	assert.Contains(t, got, "query failed")
	assert.Contains(t, got, "assert.AnError")
}

func TestErrorImplementsErrorInterface(t *testing.T) {
	var err error = &Error{Code: ErrNotFound, Message: "test"}
	assert.Implements(t, (*error)(nil), err)
}

func TestGRPCStatusInvalidInput(t *testing.T) {
	err := &Error{Code: ErrInvalidInput, Message: "bad field"}
	st := err.GRPCStatus()
	require.NotNil(t, st)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "bad field", st.Message())
}

func TestGRPCStatusValidation(t *testing.T) {
	err := &Error{Code: ErrValidation, Message: "validation failed"}
	st := err.GRPCStatus()
	require.NotNil(t, st)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestGRPCStatusNotFound(t *testing.T) {
	err := &Error{Code: ErrNotFound, Message: "user not found"}
	st := err.GRPCStatus()
	require.NotNil(t, st)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "user not found", st.Message())
}

func TestGRPCStatusDatabase(t *testing.T) {
	err := &Error{Code: ErrDatabase, Message: "connection lost"}
	st := err.GRPCStatus()
	require.NotNil(t, st)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "connection lost", st.Message())
}

func TestGRPCStatusConcurrentMod(t *testing.T) {
	err := &Error{Code: ErrConcurrentMod, Message: "version mismatch"}
	st := err.GRPCStatus()
	require.NotNil(t, st)
	assert.Equal(t, codes.Aborted, st.Code())
	assert.Equal(t, "version mismatch", st.Message())
}

func TestGRPCStatusUnknown(t *testing.T) {
	err := &Error{Code: "SOME_OTHER", Message: "mystery"}
	st := err.GRPCStatus()
	require.NotNil(t, st)
	assert.Equal(t, codes.Unknown, st.Code())
	assert.Equal(t, "mystery", st.Message())
}
