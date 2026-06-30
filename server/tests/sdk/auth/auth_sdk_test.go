package auth_test

import (
	"context"
	"testing"
	"time"

	"miqro-skillhub/server/sdk/skillhub/auth"
)

// ---- Mock Repositories ----

type mockUserRepo struct {
	users map[string]auth.UserAccount
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]auth.UserAccount)}
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*auth.UserAccount, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return &u, nil
}

func (m *mockUserRepo) FindByIDs(ctx context.Context, ids []string) ([]auth.UserAccount, error) {
	var result []auth.UserAccount
	for _, id := range ids {
		if u, ok := m.users[id]; ok {
			result = append(result, u)
		}
	}
	return result, nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*auth.UserAccount, error) {
	for _, u := range m.users {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) Save(ctx context.Context, user auth.UserAccount) (auth.UserAccount, error) {
	user.UpdatedAt = time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = user.UpdatedAt
	}
	m.users[user.ID] = user
	return user, nil
}

type mockCredRepo struct {
	creds      map[string]auth.LocalCredential
	credsByUID map[string]auth.LocalCredential
}

func newMockCredRepo() *mockCredRepo {
	return &mockCredRepo{
		creds:      make(map[string]auth.LocalCredential),
		credsByUID: make(map[string]auth.LocalCredential),
	}
}

func (m *mockCredRepo) Save(ctx context.Context, cred auth.LocalCredential) (auth.LocalCredential, error) {
	cred.UpdatedAt = time.Now()
	m.creds[cred.Username] = cred
	m.credsByUID[cred.UserID] = cred
	return cred, nil
}

func (m *mockCredRepo) FindByUserID(ctx context.Context, userID string) (*auth.LocalCredential, error) {
	c, ok := m.credsByUID[userID]
	if !ok {
		return nil, nil
	}
	return &c, nil
}

func (m *mockCredRepo) FindByUsername(ctx context.Context, username string) (*auth.LocalCredential, error) {
	c, ok := m.creds[username]
	if !ok {
		return nil, nil
	}
	return &c, nil
}

type mockIdentityBindingRepo struct {
	bindings map[int64]auth.IdentityBinding
	nextID   int64
}

func newMockIdentityBindingRepo() *mockIdentityBindingRepo {
	return &mockIdentityBindingRepo{bindings: make(map[int64]auth.IdentityBinding), nextID: 0}
}

func (m *mockIdentityBindingRepo) Save(ctx context.Context, binding auth.IdentityBinding) (auth.IdentityBinding, error) {
	if binding.ID == 0 {
		m.nextID++
		binding.ID = m.nextID
	}
	binding.UpdatedAt = time.Now()
	if binding.CreatedAt.IsZero() {
		binding.CreatedAt = binding.UpdatedAt
	}
	m.bindings[binding.ID] = binding
	return binding, nil
}

func (m *mockIdentityBindingRepo) FindByProviderAndSubject(ctx context.Context, providerCode string, subject string) (*auth.IdentityBinding, error) {
	for _, b := range m.bindings {
		if b.ProviderCode == providerCode && b.Subject == subject {
			return &b, nil
		}
	}
	return nil, nil
}

func (m *mockIdentityBindingRepo) FindByUserID(ctx context.Context, userID string) ([]auth.IdentityBinding, error) {
	var result []auth.IdentityBinding
	for _, b := range m.bindings {
		if b.UserID == userID {
			result = append(result, b)
		}
	}
	return result, nil
}

type mockApiTokenRepo struct {
	tokens map[int64]auth.ApiToken
	nextID int64
}

func newMockApiTokenRepo() *mockApiTokenRepo {
	return &mockApiTokenRepo{tokens: make(map[int64]auth.ApiToken), nextID: 0}
}

func (m *mockApiTokenRepo) Save(ctx context.Context, token auth.ApiToken) (auth.ApiToken, error) {
	if token.ID == 0 {
		m.nextID++
		token.ID = m.nextID
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}
	m.tokens[token.ID] = token
	return token, nil
}

func (m *mockApiTokenRepo) FindByID(ctx context.Context, id int64) (*auth.ApiToken, error) {
	t, ok := m.tokens[id]
	if !ok {
		return nil, nil
	}
	return &t, nil
}

func (m *mockApiTokenRepo) FindByTokenHash(ctx context.Context, hash string) (*auth.ApiToken, error) {
	for _, t := range m.tokens {
		if t.TokenHash == hash {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockApiTokenRepo) FindByUserID(ctx context.Context, userID string) ([]auth.ApiToken, error) {
	var result []auth.ApiToken
	for _, t := range m.tokens {
		if t.UserID == userID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockApiTokenRepo) FindActiveByName(ctx context.Context, userID string, name string) (*auth.ApiToken, error) {
	for _, t := range m.tokens {
		if t.UserID == userID && t.Name == name && t.RevokedAt == nil {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockApiTokenRepo) UpdateLastUsed(ctx context.Context, id int64) error {
	t, ok := m.tokens[id]
	if !ok {
		return nil
	}
	now := time.Now()
	t.LastUsedAt = &now
	m.tokens[id] = t
	return nil
}

func (m *mockApiTokenRepo) UpdateExpiration(ctx context.Context, id int64, expiresAt *time.Time) error {
	t, ok := m.tokens[id]
	if !ok {
		return nil
	}
	t.ExpiresAt = expiresAt
	m.tokens[id] = t
	return nil
}

func (m *mockApiTokenRepo) Revoke(ctx context.Context, id int64) error {
	t, ok := m.tokens[id]
	if !ok {
		return nil
	}
	now := time.Now()
	t.RevokedAt = &now
	m.tokens[id] = t
	return nil
}

type mockUserRoleBindingRepo struct {
	bindings map[int64]auth.UserRoleBinding
	nextID   int64
}

func newMockUserRoleBindingRepo() *mockUserRoleBindingRepo {
	return &mockUserRoleBindingRepo{bindings: make(map[int64]auth.UserRoleBinding), nextID: 0}
}

func (m *mockUserRoleBindingRepo) Save(ctx context.Context, binding auth.UserRoleBinding) (auth.UserRoleBinding, error) {
	m.nextID++
	binding.ID = m.nextID
	m.bindings[binding.ID] = binding
	return binding, nil
}

func (m *mockUserRoleBindingRepo) FindByUserID(ctx context.Context, userID string) ([]auth.UserRoleBinding, error) {
	var result []auth.UserRoleBinding
	for _, b := range m.bindings {
		if b.UserID == userID {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *mockUserRoleBindingRepo) DeleteByUserID(ctx context.Context, userID string) error {
	for id, b := range m.bindings {
		if b.UserID == userID {
			delete(m.bindings, id)
		}
	}
	return nil
}

type mockAccountMergeRequestRepo struct {
	requests map[int64]auth.AccountMergeRequest
	nextID   int64
}

func newMockAccountMergeRequestRepo() *mockAccountMergeRequestRepo {
	return &mockAccountMergeRequestRepo{requests: make(map[int64]auth.AccountMergeRequest), nextID: 0}
}

func (m *mockAccountMergeRequestRepo) Save(ctx context.Context, req auth.AccountMergeRequest) (auth.AccountMergeRequest, error) {
	if req.ID == 0 {
		m.nextID++
		req.ID = m.nextID
	}
	m.requests[req.ID] = req
	return req, nil
}

func (m *mockAccountMergeRequestRepo) FindByID(ctx context.Context, id int64) (*auth.AccountMergeRequest, error) {
	r, ok := m.requests[id]
	if !ok {
		return nil, nil
	}
	return &r, nil
}

func (m *mockAccountMergeRequestRepo) FindPendingBySecondaryUserID(ctx context.Context, secondaryUserID string) (*auth.AccountMergeRequest, error) {
	for _, r := range m.requests {
		if r.SecondaryUserID == secondaryUserID && r.Status == "PENDING" {
			return &r, nil
		}
	}
	return nil, nil
}

func (m *mockAccountMergeRequestRepo) Update(ctx context.Context, req *auth.AccountMergeRequest) error {
	if req == nil || req.ID == 0 {
		return nil
	}
	m.requests[req.ID] = *req
	return nil
}

// ---- Helper ----

func newTestAccountMergeService() (*auth.AccountMergeService, *mockUserRepo, *mockCredRepo, *mockIdentityBindingRepo, *mockApiTokenRepo, *mockUserRoleBindingRepo, *mockAccountMergeRequestRepo) {
	userRepo := newMockUserRepo()
	credRepo := newMockCredRepo()
	identityBindingRepo := newMockIdentityBindingRepo()
	tokenRepo := newMockApiTokenRepo()
	userRoleBindingRepo := newMockUserRoleBindingRepo()
	mergeRepo := newMockAccountMergeRequestRepo()

	svc := auth.NewAccountMergeService(
		userRepo, credRepo, identityBindingRepo, tokenRepo, userRoleBindingRepo, mergeRepo, nil,
	)
	return svc, userRepo, credRepo, identityBindingRepo, tokenRepo, userRoleBindingRepo, mergeRepo
}

// ---- TestConfirmMergeSuccess ----

func TestConfirmMergeSuccess(t *testing.T) {
	svc, userRepo, _, identityBindingRepo, tokenRepo, userRoleBindingRepo, mergeRepo := newTestAccountMergeService()
	ctx := context.Background()

	// Create primary user.
	primary := auth.UserAccount{
		ID:          "usr_primary",
		DisplayName: "Primary",
		Email:       "primary@example.com",
		Status:      "ACTIVE",
	}
	if _, err := userRepo.Save(ctx, primary); err != nil {
		t.Fatalf("save primary: %v", err)
	}

	// Create secondary user.
	secondary := auth.UserAccount{
		ID:          "usr_secondary",
		DisplayName: "Secondary",
		Email:       "secondary@example.com",
		Status:      "ACTIVE",
	}
	if _, err := userRepo.Save(ctx, secondary); err != nil {
		t.Fatalf("save secondary: %v", err)
	}

	// Add identity binding to secondary.
	binding := auth.IdentityBinding{
		UserID:       "usr_secondary",
		ProviderCode: "github",
		Subject:      "12345",
		LoginName:    "sec_user",
	}
	savedBinding, err := identityBindingRepo.Save(ctx, binding)
	if err != nil {
		t.Fatalf("save identity binding: %v", err)
	}

	// Add API token to secondary.
	token := auth.ApiToken{
		UserID:      "usr_secondary",
		SubjectType: "USER",
		SubjectID:   "usr_secondary",
		Name:        "test-token",
		TokenPrefix: "sk_test12",
		TokenHash:   "abc123",
		ScopeJSON:   `["skill:read"]`,
	}
	savedToken, err := tokenRepo.Save(ctx, token)
	if err != nil {
		t.Fatalf("save token: %v", err)
	}

	// Add role binding to secondary.
	roleBinding := auth.UserRoleBinding{
		UserID: "usr_secondary",
		RoleID: 2, // e.g., "EDITOR"
	}
	if _, err := userRoleBindingRepo.Save(ctx, roleBinding); err != nil {
		t.Fatalf("save role binding: %v", err)
	}

	// Initiate merge from primary to secondary.
	result, err := svc.InitiateMerge(ctx, "usr_primary", "usr_secondary")
	if err != nil {
		t.Fatalf("InitiateMerge failed: %v", err)
	}
	if result.RequestID == 0 {
		t.Fatal("expected non-zero request ID")
	}
	if result.RawToken == "" {
		t.Fatal("expected non-empty raw token")
	}

	// Confirm merge with correct raw token.
	err = svc.ConfirmMerge(ctx, result.RequestID, "usr_primary", result.RawToken)
	if err != nil {
		t.Fatalf("ConfirmMerge failed: %v", err)
	}

	// Verify merge request is COMPLETED.
	req, err := mergeRepo.FindByID(ctx, result.RequestID)
	if err != nil {
		t.Fatalf("find merge request: %v", err)
	}
	if req == nil {
		t.Fatal("expected merge request to exist")
	}
	if req.Status != "COMPLETED" {
		t.Fatalf("expected status COMPLETED, got %s", req.Status)
	}
	if req.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}

	// Verify secondary user is MERGED.
	secUser, err := userRepo.FindByID(ctx, "usr_secondary")
	if err != nil {
		t.Fatalf("find secondary: %v", err)
	}
	if secUser == nil {
		t.Fatal("expected secondary user to exist")
	}
	if secUser.Status != "MERGED" {
		t.Fatalf("expected secondary status MERGED, got %s", secUser.Status)
	}
	if secUser.MergedToUserID == nil || *secUser.MergedToUserID != "usr_primary" {
		t.Fatalf("expected MergedToUserID to be usr_primary, got %v", secUser.MergedToUserID)
	}

	// Verify identity binding reassigned to primary.
	bindings, err := identityBindingRepo.FindByUserID(ctx, "usr_primary")
	if err != nil {
		t.Fatalf("find identity bindings: %v", err)
	}
	foundBinding := false
	for _, b := range bindings {
		if b.ID == savedBinding.ID && b.UserID == "usr_primary" {
			foundBinding = true
			break
		}
	}
	if !foundBinding {
		t.Fatal("expected identity binding to be reassigned to primary")
	}

	// Verify API token reassigned to primary.
	tokens, err := tokenRepo.FindByUserID(ctx, "usr_primary")
	if err != nil {
		t.Fatalf("find tokens: %v", err)
	}
	foundToken := false
	for _, tk := range tokens {
		if tk.ID == savedToken.ID && tk.UserID == "usr_primary" {
			foundToken = true
			break
		}
	}
	if !foundToken {
		t.Fatal("expected API token to be reassigned to primary")
	}

	// Verify role bindings merged to primary.
	primaryBindings, err := userRoleBindingRepo.FindByUserID(ctx, "usr_primary")
	if err != nil {
		t.Fatalf("find primary role bindings: %v", err)
	}
	foundRole := false
	for _, b := range primaryBindings {
		if b.RoleID == 2 {
			foundRole = true
			break
		}
	}
	if !foundRole {
		t.Fatal("expected role ID 2 to be merged to primary")
	}

	// Verify secondary role bindings deleted.
	secondaryBindings, err := userRoleBindingRepo.FindByUserID(ctx, "usr_secondary")
	if err != nil {
		t.Fatalf("find secondary role bindings: %v", err)
	}
	if len(secondaryBindings) > 0 {
		t.Fatalf("expected secondary role bindings to be empty, got %d", len(secondaryBindings))
	}

	// Verify secondary identity bindings are now empty.
	secBindings, err := identityBindingRepo.FindByUserID(ctx, "usr_secondary")
	if err != nil {
		t.Fatalf("find secondary identity bindings: %v", err)
	}
	if len(secBindings) > 0 {
		t.Fatalf("expected secondary identity bindings to be empty, got %d", len(secBindings))
	}
}

// ---- TestConfirmMergeWrongToken ----

func TestConfirmMergeWrongToken(t *testing.T) {
	svc, userRepo, _, _, _, _, _ := newTestAccountMergeService()
	ctx := context.Background()

	primary := auth.UserAccount{ID: "usr_primary", DisplayName: "Primary", Status: "ACTIVE"}
	secondary := auth.UserAccount{ID: "usr_secondary", DisplayName: "Secondary", Status: "ACTIVE"}
	userRepo.Save(ctx, primary)
	userRepo.Save(ctx, secondary)

	result, err := svc.InitiateMerge(ctx, "usr_primary", "usr_secondary")
	if err != nil {
		t.Fatalf("InitiateMerge failed: %v", err)
	}

	err = svc.ConfirmMerge(ctx, result.RequestID, "usr_primary", "wrong-token-value")
	if err == nil {
		t.Fatal("expected error for wrong token")
	}
}

// ---- TestConfirmMergeExpiredToken ----

func TestConfirmMergeExpiredToken(t *testing.T) {
	svc, userRepo, _, _, _, _, mergeRepo := newTestAccountMergeService()
	ctx := context.Background()

	primary := auth.UserAccount{ID: "usr_primary", DisplayName: "Primary", Status: "ACTIVE"}
	secondary := auth.UserAccount{ID: "usr_secondary", DisplayName: "Secondary", Status: "ACTIVE"}
	userRepo.Save(ctx, primary)
	userRepo.Save(ctx, secondary)

	result, err := svc.InitiateMerge(ctx, "usr_primary", "usr_secondary")
	if err != nil {
		t.Fatalf("InitiateMerge failed: %v", err)
	}

	// Set token expiration to the past.
	req, err := mergeRepo.FindByID(ctx, result.RequestID)
	if err != nil {
		t.Fatalf("find merge request: %v", err)
	}
	past := time.Now().Add(-1 * time.Hour)
	req.TokenExpiresAt = &past
	mergeRepo.Save(ctx, *req)

	err = svc.ConfirmMerge(ctx, result.RequestID, "usr_primary", result.RawToken)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

// ---- TestConfirmMergeWrongStatus ----

func TestConfirmMergeWrongStatus(t *testing.T) {
	svc, userRepo, _, _, _, _, mergeRepo := newTestAccountMergeService()
	ctx := context.Background()

	primary := auth.UserAccount{ID: "usr_primary", DisplayName: "Primary", Status: "ACTIVE"}
	secondary := auth.UserAccount{ID: "usr_secondary", DisplayName: "Secondary", Status: "ACTIVE"}
	userRepo.Save(ctx, primary)
	userRepo.Save(ctx, secondary)

	result, err := svc.InitiateMerge(ctx, "usr_primary", "usr_secondary")
	if err != nil {
		t.Fatalf("InitiateMerge failed: %v", err)
	}

	// Set status to COMPLETED (not PENDING/VERIFIED).
	req, err := mergeRepo.FindByID(ctx, result.RequestID)
	if err != nil {
		t.Fatalf("find merge request: %v", err)
	}
	req.Status = "COMPLETED"
	mergeRepo.Save(ctx, *req)

	err = svc.ConfirmMerge(ctx, result.RequestID, "usr_primary", result.RawToken)
	if err == nil {
		t.Fatal("expected error for wrong status")
	}
}

// ---- TestConfirmMergeCredentialConflict ----

func TestConfirmMergeCredentialConflict(t *testing.T) {
	svc, userRepo, credRepo, _, _, _, _ := newTestAccountMergeService()
	ctx := context.Background()

	primary := auth.UserAccount{ID: "usr_primary", DisplayName: "Primary", Status: "ACTIVE"}
	secondary := auth.UserAccount{ID: "usr_secondary", DisplayName: "Secondary", Status: "ACTIVE"}
	userRepo.Save(ctx, primary)
	userRepo.Save(ctx, secondary)

	// Give both users local credentials.
	credRepo.Save(ctx, auth.LocalCredential{UserID: "usr_primary", Username: "primary_user", PasswordHash: "hash1"})
	credRepo.Save(ctx, auth.LocalCredential{UserID: "usr_secondary", Username: "secondary_user", PasswordHash: "hash2"})

	_, err := svc.InitiateMerge(ctx, "usr_primary", "usr_secondary")
	if err == nil {
		t.Fatal("expected error when both users have credentials")
	}
}

// ---- TestConfirmMergeCredentialConflictAtConfirm ----

func TestConfirmMergeCredentialConflictAtConfirm(t *testing.T) {
	svc, userRepo, credRepo, _, _, _, _ := newTestAccountMergeService()
	ctx := context.Background()

	primary := auth.UserAccount{ID: "usr_primary", DisplayName: "Primary", Status: "ACTIVE"}
	secondary := auth.UserAccount{ID: "usr_secondary", DisplayName: "Secondary", Status: "ACTIVE"}
	userRepo.Save(ctx, primary)
	userRepo.Save(ctx, secondary)

	// No credentials at init time (allowed). Add credentials to both users after initiation.
	result, err := svc.InitiateMerge(ctx, "usr_primary", "usr_secondary")
	if err != nil {
		t.Fatalf("InitiateMerge failed: %v", err)
	}

	// Add credentials to both users after initiation so they conflict at confirm time.
	credRepo.Save(ctx, auth.LocalCredential{UserID: "usr_primary", Username: "primary_user", PasswordHash: "hash1"})
	credRepo.Save(ctx, auth.LocalCredential{UserID: "usr_secondary", Username: "secondary_user", PasswordHash: "hash2"})

	err = svc.ConfirmMerge(ctx, result.RequestID, "usr_primary", result.RawToken)
	if err == nil {
		t.Fatal("expected error when both users have credentials at confirm time")
	}
}

// ---- TestUpdateExpiration ----

type mockApiTokenRepoForExpiration struct {
	mockApiTokenRepo
}

func (m *mockApiTokenRepoForExpiration) Save(ctx context.Context, token auth.ApiToken) (auth.ApiToken, error) {
	if token.ID == 0 {
		m.nextID++
		token.ID = m.nextID
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}
	m.tokens[token.ID] = token
	return token, nil
}

func newTestApiTokenService() (*auth.ApiTokenService, *mockApiTokenRepoForExpiration) {
	repo := &mockApiTokenRepoForExpiration{*newMockApiTokenRepo()}
	svc := auth.NewApiTokenService(repo)
	return svc, repo
}

func TestUpdateExpiration(t *testing.T) {
	svc, repo := newTestApiTokenService()
	ctx := context.Background()

	// Create a token.
	token := auth.ApiToken{
		UserID:      "usr_test",
		SubjectType: "USER",
		SubjectID:   "usr_test",
		Name:        "test-token",
		TokenPrefix: "sk_test12",
		TokenHash:   "abc123",
		ScopeJSON:   `["skill:read"]`,
	}
	saved, err := repo.Save(ctx, token)
	if err != nil {
		t.Fatalf("save token: %v", err)
	}

	// Update expiration.
	futureTime := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	err = svc.UpdateExpiration(ctx, saved.ID, "usr_test", futureTime)
	if err != nil {
		t.Fatalf("UpdateExpiration failed: %v", err)
	}

	// Verify expiration was updated.
	updated, err := repo.FindByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("find token: %v", err)
	}
	if updated == nil {
		t.Fatal("expected token to exist")
	}
	if updated.ExpiresAt == nil {
		t.Fatal("expected non-nil ExpiresAt")
	}
}

// ---- TestUpdateExpirationRevokedToken ----

func TestUpdateExpirationRevokedToken(t *testing.T) {
	svc, repo := newTestApiTokenService()
	ctx := context.Background()

	token := auth.ApiToken{
		UserID:      "usr_test",
		SubjectType: "USER",
		SubjectID:   "usr_test",
		Name:        "test-token",
		TokenPrefix: "sk_test12",
		TokenHash:   "abc123",
		ScopeJSON:   `["skill:read"]`,
	}
	saved, err := repo.Save(ctx, token)
	if err != nil {
		t.Fatalf("save token: %v", err)
	}

	// Revoke the token.
	if err := repo.Revoke(ctx, saved.ID); err != nil {
		t.Fatalf("revoke token: %v", err)
	}

	// Attempt to update expiration on revoked token.
	futureTime := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	err = svc.UpdateExpiration(ctx, saved.ID, "usr_test", futureTime)
	if err == nil {
		t.Fatal("expected error for revoked token")
	}
}

// ---- TestUpdateExpirationWrongUser ----

func TestUpdateExpirationWrongUser(t *testing.T) {
	svc, repo := newTestApiTokenService()
	ctx := context.Background()

	token := auth.ApiToken{
		UserID:      "usr_test",
		SubjectType: "USER",
		SubjectID:   "usr_test",
		Name:        "test-token",
		TokenPrefix: "sk_test12",
		TokenHash:   "abc123",
		ScopeJSON:   `["skill:read"]`,
	}
	saved, err := repo.Save(ctx, token)
	if err != nil {
		t.Fatalf("save token: %v", err)
	}

	// Attempt to update expiration by a different user.
	futureTime := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	err = svc.UpdateExpiration(ctx, saved.ID, "usr_foreign", futureTime)
	if err == nil {
		t.Fatal("expected error for foreign user")
	}
}
