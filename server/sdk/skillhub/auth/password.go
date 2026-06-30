package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"math/big"
	"regexp"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword returns a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword compares a password against a bcrypt hash.
// Uses constant-time comparison to mitigate timing attacks.
func CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// PasswordPolicy defines password validation rules.
type PasswordPolicy struct {
	MinLength    int // default 8
	MaxLength    int // default 128
	MinCharTypes int // default 3 (lowercase, uppercase, digit, symbol)
}

// DefaultPasswordPolicy returns the standard policy.
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{MinLength: 8, MaxLength: 128, MinCharTypes: 3}
}

// Validate checks a password against the policy.
// Returns nil if valid, or an error describing the first violation.
func (p PasswordPolicy) Validate(password string) error {
	if len(password) < p.MinLength {
		return fmt.Errorf("error.auth.local.password.tooShort")
	}
	if len(password) > p.MaxLength {
		return fmt.Errorf("error.auth.local.password.tooLong")
	}

	types := 0
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSymbol := false

	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			hasSymbol = true
		}
	}

	if hasLower {
		types++
	}
	if hasUpper {
		types++
	}
	if hasDigit {
		types++
	}
	if hasSymbol {
		types++
	}

	if types < p.MinCharTypes {
		return fmt.Errorf("error.auth.local.password.tooWeak")
	}

	return nil
}

// DummyHash is a pre-computed bcrypt hash of the string "skillhub-local-auth-dummy".
// Used for timing-attack mitigation when no credential is found during login.
const DummyHash = "$2a$12$8Q/2o2A0V.b18G2DutV4c.s5zZxH6MECM7tP8mYv6b6Q6x6o9v3vu"

// GenerateSecureCode returns a cryptographically random 6-digit numeric code as a string.
func GenerateSecureCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("auth: generate code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

var (
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,64}$`)
	emailPattern    = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)
)

// NormalizeUsername trims and lowercases a username. Returns error if format is invalid.
func NormalizeUsername(raw string) (string, error) {
	// Manual trim + lowercase without importing strings for trim
	s := raw
	// Trim spaces
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	// Lowercase
	lowered := []byte(s)
	for i, b := range lowered {
		if b >= 'A' && b <= 'Z' {
			lowered[i] = b + 32
		}
	}
	normalized := string(lowered)
	if !usernamePattern.MatchString(normalized) {
		return "", fmt.Errorf("error.auth.local.username.invalid")
	}
	return normalized, nil
}

// NormalizeEmail trims and lowercases an email. Returns empty string and nil for blank input.
func NormalizeEmail(raw string) (string, error) {
	s := raw
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	if s == "" {
		return "", nil
	}
	lowered := []byte(s)
	for i, b := range lowered {
		if b >= 'A' && b <= 'Z' {
			lowered[i] = b + 32
		}
	}
	return string(lowered), nil
}

// DummyPasswordHash is exposed for timing-attack mitigation in login.
func DummyPasswordHash() string { return DummyHash }

// ConstantTimeByteCompare compares two byte slices in constant time.
func ConstantTimeByteCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}
