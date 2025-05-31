package errors

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInternalServer     = errors.New("internal server error")
	ErrBadRequest         = errors.New("bad request")
	ErrServiceUnavailable = errors.New("service unavailable")
)

// Error represents a custom error type
type Error struct {
	Code    string
	Message string
	Err     error
}

// Error returns the error message
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error
func (e *Error) Unwrap() error {
	return e.Err
}

// New creates a new error with a message
func New(message string) error {
	return &Error{
		Message: message,
	}
}

// Wrap wraps an error with additional message
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return &Error{
		Message: message,
		Err:     err,
	}
}

// WrapWithCode wraps an error with a code and message
func WrapWithCode(err error, code, message string) error {
	if err == nil {
		return nil
	}
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Is reports whether any error in err's chain matches target
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// GetCode returns the error code if it exists
func GetCode(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// GetMessage returns the error message
func GetMessage(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Message
	}
	return err.Error()
}

// IsNotFound returns true if the error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized returns true if the error is an unauthorized error
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden returns true if the error is a forbidden error
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsInternalServer returns true if the error is an internal server error
func IsInternalServer(err error) bool {
	return errors.Is(err, ErrInternalServer)
}

// IsBadRequest returns true if the error is a bad request error
func IsBadRequest(err error) bool {
	return errors.Is(err, ErrBadRequest)
}

// IsServiceUnavailable returns true if the error is a service unavailable error
func IsServiceUnavailable(err error) bool {
	return errors.Is(err, ErrServiceUnavailable)
}
