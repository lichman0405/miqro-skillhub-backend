// Package errors defines the typed error model used across all SDK
// packages so that HTTP, CLI, workers, and tests can map errors
// consistently.
//
// Source reference:
//
//	com.iflytek.skillhub.domain.shared.exception
//	  LocalizedDomainException / LocalizedMessage
//	  DomainBadRequestException    (400)
//	  DomainForbiddenException     (403)
//	  DomainNotFoundException      (404)
package errors

import (
	"fmt"
)

// Code is a stable machine-readable error kind.
type Code string

const (
	// ErrBadRequest indicates the caller's input failed business validation.
	ErrBadRequest Code = "bad_request"

	// ErrForbidden indicates the caller lacks permission for the action.
	ErrForbidden Code = "forbidden"

	// ErrNotFound indicates a required business entity does not exist.
	ErrNotFound Code = "not_found"

	// ErrConflict indicates a conflicting resource or state.
	ErrConflict Code = "conflict"

	// ErrUnauthorized indicates the caller's credentials are missing,
	// invalid, or expired.
	ErrUnauthorized Code = "unauthorized"

	// ErrInternal indicates an unexpected server-side failure.
	ErrInternal Code = "internal"
)

// Error is a typed domain error carrying a machine-readable code,
// a localizable message key (preserved from the Java source where
// practical), interpolation arguments, and an optional wrapped cause.
type Error struct {
	Code       Code
	MessageKey string
	Args       []any
	Cause      error
}

// Error implements the error interface.
func (e *Error) Error() string {
	msg := fmt.Sprintf("[%s] %s", e.Code, e.MessageKey)
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

// Unwrap returns the wrapped cause so callers can use errors.Is / errors.As.
func (e *Error) Unwrap() error {
	return e.Cause
}

// New returns a new Error with the given code and message key.
func New(code Code, messageKey string, args ...any) *Error {
	return &Error{Code: code, MessageKey: messageKey, Args: args}
}

// Wrap returns a new Error that wraps an existing cause.
func Wrap(cause error, code Code, messageKey string, args ...any) *Error {
	return &Error{Code: code, MessageKey: messageKey, Args: args, Cause: cause}
}

// BadRequest is a convenience constructor for ErrBadRequest.
func BadRequest(messageKey string, args ...any) *Error {
	return New(ErrBadRequest, messageKey, args...)
}

// Forbidden is a convenience constructor for ErrForbidden.
func Forbidden(messageKey string, args ...any) *Error {
	return New(ErrForbidden, messageKey, args...)
}

// NotFound is a convenience constructor for ErrNotFound.
func NotFound(messageKey string, args ...any) *Error {
	return New(ErrNotFound, messageKey, args...)
}

// Conflict is a convenience constructor for ErrConflict.
func Conflict(messageKey string, args ...any) *Error {
	return New(ErrConflict, messageKey, args...)
}

// Unauthorized is a convenience constructor for ErrUnauthorized.
func Unauthorized(messageKey string, args ...any) *Error {
	return New(ErrUnauthorized, messageKey, args...)
}

// Internal is a convenience constructor for ErrInternal.
func Internal(messageKey string, args ...any) *Error {
	return New(ErrInternal, messageKey, args...)
}
