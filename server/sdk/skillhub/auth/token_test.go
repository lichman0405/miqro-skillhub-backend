package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"
)

// mockTokenRepo implements ApiTokenRepository for testing.
type mockTokenRepo struct {
	tokens map[int64]ApiToken
	nextID int64
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{
		tokens: make(map[int64]ApiToken),
		nextID: 1,
	}
}

func (m *mockTokenRepo) Save(ctx context.Context, token ApiToken) (ApiToken, error) {
	existing, _ := m.FindActiveByName(ctx, token.UserID, strings.ToLower(token.Name))
	if existing != nil {
		return ApiToken{}, fmt.Errorf("duplicate key value violates unique constraint")
	}
	token.ID = m.nextID
	m.nextID++
	m.tokens[token.ID] = token
	return token, nil
}

func (m *mockTokenRepo) FindByID(ctx context.Context, id int64) (*ApiToken, error) {
	t, ok := m.tokens[id]
	if !ok {
		return nil, nil
	}
	return &t, nil
}

func (m *mockTokenRepo) FindByTokenHash(ctx context.Context, hash string) (*ApiToken, error) {
	for _, t := range m.tokens {
		if t.TokenHash == hash {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockTokenRepo) FindByUserID(ctx context.Context, userID string) ([]ApiToken, error) {
	var result []ApiToken
	for _, t := range m.tokens {
		if t.UserID == userID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockTokenRepo) FindActiveByName(ctx context.Context, userID string, name string) (*ApiToken, error) {
	for _, t := range m.tokens {
		if t.UserID == userID && strings.EqualFold(t.Name, name) && t.RevokedAt == nil {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockTokenRepo) UpdateLastUsed(ctx context.Context, id int64) error {
	t, ok := m.tokens[id]
	if !ok {
		return nil
	}
	now := time.Now()
	t.LastUsedAt = &now
	m.tokens[id] = t
	return nil
}

func (m *mockTokenRepo) UpdateExpiration(ctx context.Context, id int64, expiresAt *time.Time) error {
	t, ok := m.tokens[id]
	if !ok {
		return nil
	}
	t.ExpiresAt = expiresAt
	m.tokens[id] = t
	return nil
}

func (m *mockTokenRepo) Revoke(ctx context.Context, id int64) error {
	t, ok := m.tokens[id]
	if !ok {
		return nil
	}
	now := time.Now()
	t.RevokedAt = &now
	m.tokens[id] = t
	return nil
}

func TestCreateTokenSkPrefix(t *testing.T) {
	repo := newMockTokenRepo()
	svc := NewApiTokenService(repo)
	ctx := context.Background()

	result, err := svc.CreateToken(ctx, "user-1", "MyToken", []string{"skill:read"}, nil)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !strings.HasPrefix(result.RawToken, "sk_") {
		t.Fatalf("expected token to start with 'sk_', got: %s", result.RawToken)
	}
	if result.Token.Name != "MyToken" {
		t.Fatalf("expected token name 'MyToken', got: %s", result.Token.Name)
	}
}

func TestCreateTokenReturnsRawTokenOnce(t *testing.T) {
	repo := newMockTokenRepo()
	svc := NewApiTokenService(repo)
	ctx := context.Background()

	result, err := svc.CreateToken(ctx, "user-1", "MyToken", []string{"skill:read"}, nil)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	// The raw token should be returned in the result but not stored directly.
	if result.RawToken == "" {
		t.Fatal("expected non-empty raw token")
	}
	// The stored token should only have a hash, not the raw token.
	if result.Token.TokenHash == result.RawToken {
		t.Fatal("stored token should contain a hash, not the raw token")
	}
}

func TestTokenPrefixIsFirst8Chars(t *testing.T) {
	repo := newMockTokenRepo()
	svc := NewApiTokenService(repo)
	ctx := context.Background()

	result, err := svc.CreateToken(ctx, "user-1", "MyToken", []string{"skill:read"}, nil)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}
	expectedPrefix := result.RawToken[:8]
	if result.Token.TokenPrefix != expectedPrefix {
		t.Fatalf("expected token prefix '%s', got '%s'", expectedPrefix, result.Token.TokenPrefix)
	}
}

func TestCreateTokenDuplicateName(t *testing.T) {
	repo := newMockTokenRepo()
	svc := NewApiTokenService(repo)
	ctx := context.Background()

	_, err := svc.CreateToken(ctx, "user-1", "MyToken", []string{"skill:read"}, nil)
	if err != nil {
		t.Fatalf("first CreateToken failed: %v", err)
	}

	_, err = svc.CreateToken(ctx, "user-1", "MyToken", []string{"skill:read"}, nil)
	if err == nil {
		t.Fatal("expected error for duplicate token name")
	}
}

func TestValidateTokenValid(t *testing.T) {
	repo := newMockTokenRepo()
	svc := NewApiTokenService(repo)
	ctx := context.Background()

	result, err := svc.CreateToken(ctx, "user-1", "MyToken", []string{"skill:read"}, nil)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	validated, err := svc.ValidateToken(ctx, result.RawToken)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if validated == nil {
		t.Fatal("expected non-nil validated token")
	}
	if validated.Name != "MyToken" {
		t.Fatalf("expected token name 'MyToken', got '%s'", validated.Name)
	}
}

func TestValidateTokenRevoked(t *testing.T) {
	repo := newMockTokenRepo()
	svc := NewApiTokenService(repo)
	ctx := context.Background()

	result, err := svc.CreateToken(ctx, "user-1", "MyToken", []string{"skill:read"}, nil)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	err = svc.RevokeToken(ctx, result.Token.ID, "user-1")
	if err != nil {
		t.Fatalf("RevokeToken failed: %v", err)
	}

	validated, err := svc.ValidateToken(ctx, result.RawToken)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if validated != nil {
		t.Fatal("expected nil for revoked token")
	}
}

func TestValidateTokenExpired(t *testing.T) {
	repo := newMockTokenRepo()
	svc := NewApiTokenService(repo)
	ctx := context.Background()

	pastTime := time.Now().Add(-1 * time.Hour)
	result, err := svc.CreateToken(ctx, "user-1", "MyToken", []string{"skill:read"}, &pastTime)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	validated, err := svc.ValidateToken(ctx, result.RawToken)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if validated != nil {
		t.Fatal("expected nil for expired token")
	}
}

func TestParseScopes(t *testing.T) {
	scopes := ParseScopes(`["skill:read","skill:publish"]`)
	if len(scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(scopes))
	}
	if scopes[0] != "skill:read" {
		t.Errorf("expected 'skill:read', got '%s'", scopes[0])
	}
	if scopes[1] != "skill:publish" {
		t.Errorf("expected 'skill:publish', got '%s'", scopes[1])
	}

	// Empty array.
	scopes = ParseScopes("[]")
	if len(scopes) != 0 {
		t.Fatalf("expected 0 scopes for empty array, got %d", len(scopes))
	}

	// Empty string.
	scopes = ParseScopes("")
	if scopes != nil {
		t.Fatalf("expected nil for empty string, got %v", scopes)
	}

	// Single scope.
	scopes = ParseScopes(`["token:manage"]`)
	if len(scopes) != 1 {
		t.Fatalf("expected 1 scope, got %d", len(scopes))
	}
	if scopes[0] != "token:manage" {
		t.Errorf("expected 'token:manage', got '%s'", scopes[0])
	}
}

func TestScopesToJSON(t *testing.T) {
	json := scopesToJSON([]string{"skill:read", "skill:publish"})
	if json != `["skill:read","skill:publish"]` {
		t.Fatalf("expected '[\"skill:read\",\"skill:publish\"]', got '%s'", json)
	}

	// Empty list.
	json = scopesToJSON([]string{})
	if json != "[]" {
		t.Fatalf("expected '[]', got '%s'", json)
	}

	// Nil list.
	json = scopesToJSON(nil)
	if json != "[]" {
		t.Fatalf("expected '[]', got '%s'", json)
	}
}

func TestParseExpiresAtAcceptsRFC3339(t *testing.T) {
	future := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	tm, err := ParseExpiresAt(future)
	if err != nil {
		t.Fatalf("ParseExpiresAt failed for RFC3339: %v", err)
	}
	if tm == nil {
		t.Fatal("expected non-nil time")
	}
}

func TestParseExpiresAtRejectsPastDate(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	_, err := ParseExpiresAt(past)
	if err == nil {
		t.Fatal("expected error for past date")
	}
}

func TestParseExpiresAtEmptyString(t *testing.T) {
	tm, err := ParseExpiresAt("")
	if err != nil {
		t.Fatalf("ParseExpiresAt failed for empty string: %v", err)
	}
	if tm != nil {
		t.Fatal("expected nil for empty string")
	}
}

func TestRotateTokenRevokesExistingAndCreatesNew(t *testing.T) {
	repo := newMockTokenRepo()
	svc := NewApiTokenService(repo)
	ctx := context.Background()

	// Create first token.
	first, err := svc.CreateToken(ctx, "user-1", "MyToken", []string{"skill:read"}, nil)
	if err != nil {
		t.Fatalf("first CreateToken failed: %v", err)
	}

	// Rotate it.
	second, err := svc.RotateToken(ctx, "user-1", "MyToken", []string{"skill:publish"}, nil)
	if err != nil {
		t.Fatalf("RotateToken failed: %v", err)
	}

	// First token should now be revoked.
	validated, err := svc.ValidateToken(ctx, first.RawToken)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if validated != nil {
		t.Fatal("expected nil for revoked (rotated) token")
	}

	// Second token should be valid.
	validated, err = svc.ValidateToken(ctx, second.RawToken)
	if err != nil {
		t.Fatalf("ValidateToken for new token failed: %v", err)
	}
	if validated == nil {
		t.Fatal("expected non-nil for new rotated token")
	}
	if validated.Name != "MyToken" {
		t.Fatalf("expected token name 'MyToken', got '%s'", validated.Name)
	}

	// Verify the new token has different raw token.
	if first.RawToken == second.RawToken {
		t.Fatal("rotated token should have different raw token")
	}
}

func TestTokenHashIsSHA256(t *testing.T) {
	repo := newMockTokenRepo()
	svc := NewApiTokenService(repo)
	ctx := context.Background()

	result, err := svc.CreateToken(ctx, "user-1", "MyToken", []string{"skill:read"}, nil)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	// Manually compute SHA-256 of the raw token and verify it matches.
	expectedHash := sha256.Sum256([]byte(result.RawToken))
	expectedHashHex := hex.EncodeToString(expectedHash[:])
	if result.Token.TokenHash != expectedHashHex {
		t.Fatalf("expected token hash '%s', got '%s'", expectedHashHex, result.Token.TokenHash)
	}
}
