package errors_test

import (
	"errors"
	"testing"

	skerrors "miqro-skillhub/server/sdk/skillhub/errors"
)

func TestNew(t *testing.T) {
	err := skerrors.New(skerrors.ErrBadRequest, "test.key", "arg1")
	if err.Code != skerrors.ErrBadRequest {
		t.Errorf("expected bad_request, got %s", err.Code)
	}
	if err.MessageKey != "test.key" {
		t.Errorf("expected test.key, got %s", err.MessageKey)
	}
	if len(err.Args) != 1 || err.Args[0] != "arg1" {
		t.Errorf("unexpected args: %v", err.Args)
	}
	if err.Cause != nil {
		t.Errorf("expected nil cause")
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("underlying")
	err := skerrors.Wrap(cause, skerrors.ErrInternal, "internal.key")
	if !errors.Is(err, cause) {
		t.Errorf("expected errors.Is to find cause")
	}
	if err.Cause != cause {
		t.Errorf("expected cause to be set")
	}
}

func TestConvenienceConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *skerrors.Error
		wantCode skerrors.Code
	}{
		{"BadRequest", skerrors.BadRequest("key"), skerrors.ErrBadRequest},
		{"Forbidden", skerrors.Forbidden("key"), skerrors.ErrForbidden},
		{"NotFound", skerrors.NotFound("key"), skerrors.ErrNotFound},
		{"Conflict", skerrors.Conflict("key"), skerrors.ErrConflict},
		{"Unauthorized", skerrors.Unauthorized("key"), skerrors.ErrUnauthorized},
		{"Internal", skerrors.Internal("key"), skerrors.ErrInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("expected %s, got %s", tt.wantCode, tt.err.Code)
			}
		})
	}
}

func TestErrorImplementsError(t *testing.T) {
	var _ error = (*skerrors.Error)(nil)
}

func TestErrorMessage(t *testing.T) {
	err := skerrors.New(skerrors.ErrNotFound, "skill.not.found", "slug-1")
	msg := err.Error()
	if msg == "" {
		t.Errorf("expected non-empty error message")
	}
}

func TestErrorMessageWithCause(t *testing.T) {
	cause := errors.New("db timeout")
	err := skerrors.Wrap(cause, skerrors.ErrInternal, "internal")
	msg := err.Error()
	if msg == "" {
		t.Errorf("expected non-empty error message")
	}
}
