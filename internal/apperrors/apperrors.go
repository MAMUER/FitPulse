package apperrors

import "fmt"

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func WithMessage(err error, message string) *AppError {
	return &AppError{
		Code:    "INTERNAL",
		Message: message,
		Err:     err,
	}
}

func NotFound(message string) *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: message,
	}
}

func Unauthorized(message string) *AppError {
	return &AppError{
		Code:    "UNAUTHORIZED",
		Message: message,
	}
}

func Forbidden(message string) *AppError {
	return &AppError{
		Code:    "FORBIDDEN",
		Message: message,
	}
}

func InvalidArgument(message string) *AppError {
	return &AppError{
		Code:    "INVALID_ARGUMENT",
		Message: message,
	}
}

func Conflict(message string) *AppError {
	return &AppError{
		Code:    "CONFLICT",
		Message: message,
	}
}

func Internal(message string, err error) *AppError {
	return &AppError{
		Code:    "INTERNAL",
		Message: message,
		Err:     err,
	}
}

func Unavailable(message string) *AppError {
	return &AppError{
		Code:    "UNAVAILABLE",
		Message: message,
	}
}

func Validation(message string) *AppError {
	return &AppError{
		Code:    "VALIDATION",
		Message: message,
	}
}

func RateLimited(message string) *AppError {
	return &AppError{
		Code:    "RATE_LIMITED",
		Message: message,
	}
}

func Wrap(err error, code, message string) *AppError {
	if err == nil {
		return &AppError{Code: code, Message: message}
	}
	return &AppError{Code: code, Message: message, Err: err}
}

func Code(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return "INTERNAL"
}

func Message(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Message
	}
	return "internal error"
}

func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

func GRPCCode(err error) string {
	switch Code(err) {
	case "NOT_FOUND":
		return "NotFound"
	case "UNAUTHORIZED":
		return "Unauthenticated"
	case "FORBIDDEN":
		return "PermissionDenied"
	case "INVALID_ARGUMENT", "VALIDATION":
		return "InvalidArgument"
	case "CONFLICT":
		return "AlreadyExists"
	case "UNAVAILABLE":
		return "Unavailable"
	case "RATE_LIMITED":
		return "ResourceExhausted"
	default:
		return "Internal"
	}
}

func HTTPStatus(err error) int {
	switch Code(err) {
	case "NOT_FOUND":
		return 404
	case "UNAUTHORIZED":
		return 401
	case "FORBIDDEN":
		return 403
	case "INVALID_ARGUMENT", "VALIDATION":
		return 400
	case "CONFLICT":
		return 409
	case "UNAVAILABLE":
		return 503
	case "RATE_LIMITED":
		return 429
	default:
		return 500
	}
}

func ToGRPCStatus(err error) string {
	return GRPCCode(err)
}

func Format(err error) string {
	if appErr, ok := err.(*AppError); ok {
		if appErr.Err != nil {
			return fmt.Sprintf("%s: %s", appErr.Message, appErr.Err.Error())
		}
		return appErr.Message
	}
	return err.Error()
}
