package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	TokenPrefix  = "sk_"
	TokenBytes   = 32
	MaxTokenName = 64
)

// ApiTokenService provides API token lifecycle operations.
type ApiTokenService struct {
	repo ApiTokenRepository
}

// NewApiTokenService creates a new ApiTokenService.
func NewApiTokenService(repo ApiTokenRepository) *ApiTokenService {
	return &ApiTokenService{repo: repo}
}

// TokenCreateResult holds the raw token (shown once) and the persisted entity.
type TokenCreateResult struct {
	RawToken string
	Token    ApiToken
}

// CreateToken generates a new API token for the given user.
func (s *ApiTokenService) CreateToken(ctx context.Context, userID string, name string, scopes []string, expiresAt *time.Time) (*TokenCreateResult, error) {
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > MaxTokenName {
		return nil, fmt.Errorf("error.token.name.invalid")
	}

	// Check uniqueness: active token name per user (case-insensitive).
	existing, _ := s.repo.FindActiveByName(ctx, userID, strings.ToLower(name))
	if existing != nil {
		return nil, fmt.Errorf("error.token.name.duplicate")
	}

	// Generate 32 random bytes.
	randomBytes := make([]byte, TokenBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("auth: token random: %w", err)
	}

	// Encode raw token: sk_ + base64url (no padding).
	rawToken := TokenPrefix + base64.RawURLEncoding.EncodeToString(randomBytes)

	// SHA-256 hash for storage (lowercase hex).
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Display prefix: first 8 chars of raw token.
	prefix := rawToken
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}

	// Serialize scopes to JSON array string.
	scopeJSON := scopesToJSON(scopes)

	now := time.Now()
	token := ApiToken{
		SubjectType: "USER",
		SubjectID:   userID,
		UserID:      userID,
		Name:        name,
		TokenPrefix: prefix,
		TokenHash:   tokenHash,
		ScopeJSON:   scopeJSON,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
	}

	saved, err := s.repo.Save(ctx, token)
	if err != nil {
		// Check for duplicate on second insert (race).
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return nil, fmt.Errorf("error.token.name.duplicate")
		}
		return nil, fmt.Errorf("auth: save token: %w", err)
	}

	return &TokenCreateResult{RawToken: rawToken, Token: saved}, nil
}

// RotateToken revokes the existing active token with the given name and creates a new one.
func (s *ApiTokenService) RotateToken(ctx context.Context, userID string, name string, scopes []string, expiresAt *time.Time) (*TokenCreateResult, error) {
	existing, err := s.repo.FindActiveByName(ctx, userID, strings.ToLower(name))
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if err := s.repo.Revoke(ctx, existing.ID); err != nil {
			return nil, fmt.Errorf("auth: revoke token: %w", err)
		}
	}
	return s.CreateToken(ctx, userID, name, scopes, expiresAt)
}

// ValidateToken validates a raw token string and returns the ApiToken if valid.
func (s *ApiTokenService) ValidateToken(ctx context.Context, rawToken string) (*ApiToken, error) {
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	token, err := s.repo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("auth: validate token: %w", err)
	}
	if token == nil {
		return nil, nil
	}

	now := time.Now()
	if token.RevokedAt != nil {
		return nil, nil
	}
	if token.ExpiresAt != nil && token.ExpiresAt.Before(now) {
		return nil, nil
	}

	return token, nil
}

// RevokeToken revokes a token. Idempotent: missing or foreign tokens are silently ignored.
func (s *ApiTokenService) RevokeToken(ctx context.Context, tokenID int64, userID string) error {
	token, err := s.repo.FindByID(ctx, tokenID)
	if err != nil {
		return nil // Idempotent: missing token = no error.
	}
	if token == nil || token.UserID != userID {
		return nil // Foreign token = silently ignored.
	}
	return s.repo.Revoke(ctx, tokenID)
}

// ListTokens returns all tokens for a user.
func (s *ApiTokenService) ListTokens(ctx context.Context, userID string) ([]ApiToken, error) {
	return s.repo.FindByUserID(ctx, userID)
}

// TouchLastUsed updates the last_used_at timestamp.
func (s *ApiTokenService) TouchLastUsed(ctx context.Context, tokenID int64) error {
	return s.repo.UpdateLastUsed(ctx, tokenID)
}

// ParseExpiresAt parses an expiration time string. Accepts:
//
//	RFC3339 instant, offset datetime, or legacy naive UTC timestamp.
//
// Returns nil if the string is empty.
func ParseExpiresAt(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, value); err == nil {
			now := time.Now()
			if t.Before(now) {
				return nil, fmt.Errorf("validation.token.expiresAt.future")
			}
			return &t, nil
		}
	}

	return nil, fmt.Errorf("validation.token.expiresAt.invalid")
}

// ParseScopes parses a scope JSON string into a list of scope strings.
func ParseScopes(scopeJSON string) []string {
	// Trim brackets and split.
	s := strings.TrimSpace(scopeJSON)
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		return nil
	}
	inner := s[1 : len(s)-1]
	if strings.TrimSpace(inner) == "" {
		return nil
	}

	parts := strings.Split(inner, ",")
	seen := make(map[string]bool)
	var scopes []string
	for _, p := range parts {
		scope := strings.TrimSpace(p)
		// Remove quotes.
		scope = strings.Trim(scope, "\"")
		if scope == "" {
			continue
		}
		if !seen[scope] {
			seen[scope] = true
			scopes = append(scopes, scope)
		}
	}
	return scopes
}

// scopesToJSON converts a scope list to JSON array string.
func scopesToJSON(scopes []string) string {
	if len(scopes) == 0 {
		return "[]"
	}
	quoted := make([]string, len(scopes))
	for i, s := range scopes {
		quoted[i] = `"` + s + `"`
	}
	return "[" + strings.Join(quoted, ",") + "]"
}
