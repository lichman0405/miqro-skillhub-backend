package middleware

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	sdkerror "miqro-skillhub/server/sdk/skillhub/errors"
)

// ---------------------------------------------------------------------------
// Response envelope — shared by all HTTP surfaces
// ---------------------------------------------------------------------------

// Envelope is the standard JSON response wrapper.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
}

// ErrorBody is a structured error for the HTTP response.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes an Envelope as JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Envelope{Success: true, Data: v}); err != nil {
		log.Printf("http: failed to write JSON response: %v", err)
	}
}

// WriteError maps an SDK error (or generic error) to an HTTP status and
// writes the error envelope.
func WriteError(w http.ResponseWriter, err error) {
	status, code := mapError(err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Success: false,
		Error: &ErrorBody{
			Code:    code,
			Message: err.Error(),
		},
	})
}

// mapError translates SDK errors to HTTP status codes.
func mapError(err error) (status int, code string) {
	// Auth middleware errors carry explicit status/code.
	var ae *authError
	if errors.As(err, &ae) {
		return ae.status, ae.code
	}

	var se *sdkerror.Error
	if errors.As(err, &se) {
		switch se.Code {
		case sdkerror.ErrBadRequest:
			return http.StatusBadRequest, "bad_request"
		case sdkerror.ErrUnauthorized:
			return http.StatusUnauthorized, "unauthorized"
		case sdkerror.ErrForbidden:
			return http.StatusForbidden, "forbidden"
		case sdkerror.ErrNotFound:
			return http.StatusNotFound, "not_found"
		case sdkerror.ErrConflict:
			return http.StatusConflict, "conflict"
		case sdkerror.ErrInternal:
			return http.StatusInternalServerError, "internal_error"
		}
	}

	// Fallback: inspect the error message for known patterns.
	msg := err.Error()
	if len(msg) >= 4 {
		switch {
		case matchMsgPrefix(msg, "error.", "not_found"):
			return http.StatusNotFound, "not_found"
		case matchMsgPrefix(msg, "error.", "forbidden"):
			return http.StatusForbidden, "forbidden"
		case matchMsgPrefix(msg, "error.", "unauthorized"):
			return http.StatusUnauthorized, "unauthorized"
		case matchMsgPrefix(msg, "error.", "conflict"):
			return http.StatusConflict, "conflict"
		}
	}

	return http.StatusInternalServerError, "internal_error"
}

func matchMsgPrefix(msg, prefix, kind string) bool {
	_ = kind
	return len(msg) > len(prefix) && msg[:len(prefix)] == prefix
}
