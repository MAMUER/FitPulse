// Package validator provides input validation utilities for API requests.
package validator

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NilRequestError returns a standardized error for nil requests.
func NilRequestError() error {
	return status.Error(codes.InvalidArgument, "request is nil")
}

// RequireString returns an InvalidArgument error if value is empty.
func RequireString(value, fieldName string, err error) error {
	if value == "" {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}
