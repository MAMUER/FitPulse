package apperrors

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrConflict          = errors.New("conflict")
	ErrInternal          = errors.New("internal error")
	ErrUnavailable       = errors.New("service unavailable")
	ErrValidation        = errors.New("validation failed")
	ErrRateLimited       = errors.New("rate limited")
	ErrEmailNotConfirmed = errors.New("email not confirmed")
	ErrTOTPRequired      = errors.New("totp required")
	ErrTOTPInvalid       = errors.New("totp invalid")
	ErrRefreshToken      = errors.New("refresh token invalid")
)
