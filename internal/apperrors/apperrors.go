// Package errors provides application error types and utilities.
package errors

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrorCode string

const (
	ErrInvalidInput  ErrorCode = "INVALID_INPUT"
	ErrNotFound      ErrorCode = "NOT_FOUND"
	ErrDatabase      ErrorCode = "DATABASE_ERROR"
	ErrValidation    ErrorCode = "VALIDATION_FAILED"
	ErrConcurrentMod ErrorCode = "CONCURRENT_MODIFICATION"
)

type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) GRPCStatus() *status.Status {
	// Маппинг доменных ошибок в gRPC коды
	switch e.Code {
	case ErrInvalidInput, ErrValidation:
		return status.New(codes.InvalidArgument, e.Message)
	case ErrNotFound:
		return status.New(codes.NotFound, e.Message)
	case ErrDatabase:
		return status.New(codes.Internal, e.Message)
	case ErrConcurrentMod:
		return status.New(codes.Aborted, e.Message)
	default:
		return status.New(codes.Unknown, e.Message)
	}
}
