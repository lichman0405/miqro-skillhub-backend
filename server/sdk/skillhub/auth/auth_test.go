package auth

import (
	"context"
	"testing"
	"time"
)

// ---- Mock Repositories ----

type mockUserRepo struct {
	users map[string]UserAccount
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]UserAccount)}
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*UserAccount, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return &u, nil
}

func (m *mockUserRepo) FindByIDs(ctx context.Context, ids []string) ([]UserAccount, error) {
	var result []UserAccount
	for _, id := range ids {
		if u, ok := m.users[id]; ok {
			result = append(result, u)
		}
	}
	return result, nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*UserAccount, error) {
	for _, u := range m.users {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) Save(ctx context.Context, user UserAccount) (UserAccount, error) {
	user.UpdatedAt = time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = user.UpdatedAt
	}
	m.users[user.ID] = user
	return user, nil
}

type mockCredRepo struct {
	creds      map[string]LocalCredential
	credsByUID map[string]LocalCredential
}

func newMockCredRepo() *mockCredRepo {
	return &mockCredRepo{
		creds:      make(map[string]LocalCredential),
		credsByUID: make(map[string]LocalCredential),
	}
}

func (m *mockCredRepo) Save(ctx context.Context, cred LocalCredential) (LocalCredential, error) {
	cred.UpdatedAt = time.Now()
	m.creds[cred.Username] = cred
	m.credsByUID[cred.UserID] = cred
	return cred, nil
}

func (m *mockCredRepo) FindByUserID(ctx context.Context, userID string) (*LocalCredential, error) {
	c, ok := m.credsByUID[userID]
	if !ok {
		return nil, nil
	}
	return &c, nil
}

func (m *mockCredRepo) FindByUsername(ctx context.Context, username string) (*LocalCredential, error) {
	c, ok := m.creds[username]
	if !ok {
		return nil, nil
	}
	return &c, nil
}

type mockRoleRepo struct {
	roles  map[int64]Role
	nextID int64
}

func newMockRoleRepo() *mockRoleRepo {
	return &mockRoleRepo{roles: make(map[int64]Role), nextID: 0}
}

func (m *mockRoleRepo) FindByID(ctx context.Context, id int64) (*Role, error) {
	r, ok := m.roles[id]
	if !ok {
		return nil, nil
	}
	return &r, nil
}

func (m *mockRoleRepo) FindByCode(ctx context.Context, code string) (*Role, error) {
	for _, r := range m.roles {
		if r.Code == code {
			return &r, nil
		}
	}
	return nil, nil
}

func (m *mockRoleRepo) FindAll(ctx context.Context) ([]Role, error) {
	var result []Role
	for _, r := range m.roles {
		result = append(result, r)
	}
	return result, nil
}

func (m *mockRoleRepo) Save(ctx context.Context, role Role) (Role, error) {
	if role.ID == 0 {
		m.nextID++
		role.ID = m.nextID
	}
	m.roles[role.ID] = role
	return role, nil
}

type mockPermissionRepo struct {
	perms []Permission
}

func newMockPermissionRepo() *mockPermissionRepo {
	return &mockPermissionRepo{}
}

func (m *mockPermissionRepo) FindByID(ctx context.Context, id int64) (*Permission, error) {
	for _, p := range m.perms {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, nil
}

func (m *mockPermissionRepo) FindByCode(ctx context.Context, code string) (*Permission, error) {
	for _, p := range m.perms {
		if p.Code == code {
			return &p, nil
		}
	}
	return nil, nil
}

func (m *mockPermissionRepo) FindAll(ctx context.Context) ([]Permission, error) {
	return m.perms, nil
}

type mockUserRoleBindingRepo struct {
	bindings map[int64]UserRoleBinding
	nextID   int64
}

func newMockUserRoleBindingRepo() *mockUserRoleBindingRepo {
	return &mockUserRoleBindingRepo{bindings: make(map[int64]UserRoleBinding), nextID: 0}
}

func (m *mockUserRoleBindingRepo) Save(ctx context.Context, binding UserRoleBinding) (UserRoleBinding, error) {
	m.nextID++
	binding.ID = m.nextID
	m.bindings[binding.ID] = binding
	return binding, nil
}

func (m *mockUserRoleBindingRepo) FindByUserID(ctx context.Context, userID string) ([]UserRoleBinding, error) {
	var result []UserRoleBinding
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

// ---- helper ----

func newTestLocalAuthService() (*LocalAuthService, *mockUserRepo, *mockCredRepo) {
	userRepo := newMockUserRepo()
	credRepo := newMockCredRepo()
	roleRepo := newMockRoleRepo()
	bindingRepo := newMockUserRoleBindingRepo()

	svc := NewLocalAuthService(userRepo, credRepo, bindingRepo, roleRepo)
	return svc, userRepo, credRepo
}

// ---- Tests ----

func TestRegisterValidUser(t *testing.T) {
	svc, userRepo, credRepo := newTestLocalAuthService()
	ctx := context.Background()

	result, err := svc.Register(ctx, "testuser", "test@example.com", "MyP@ssw0rd")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify user was saved.
	if result.User.ID == "" {
		t.Fatal("expected non-empty user ID")
	}
	if !isSavedUser(userRepo, result.User.ID) {
		t.Fatal("expected user to be saved in repository")
	}

	// Verify credential was saved.
	if !isSavedCredential(credRepo, "testuser") {
		t.Fatal("expected credential to be saved for username 'testuser'")
	}

	// Verify principal has default "USER" role.
	if !result.Principal.HasRole("USER") {
		t.Fatal("expected principal to have default 'USER' role")
	}
}

func TestRegisterDuplicateUsername(t *testing.T) {
	svc, _, _ := newTestLocalAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "testuser", "a@example.com", "MyP@ssw0rd")
	if err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	_, err = svc.Register(ctx, "testuser", "b@example.com", "MyP@ssw0rd")
	if err == nil {
		t.Fatal("expected error for duplicate username")
	}
}

func TestLoginWithCorrectPassword(t *testing.T) {
	svc, _, _ := newTestLocalAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "testuser", "test@example.com", "MyP@ssw0rd")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	principal, err := svc.Login(ctx, "testuser", "MyP@ssw0rd")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if principal == nil {
		t.Fatal("expected non-nil principal")
	}
	if !principal.HasRole("USER") {
		t.Fatal("expected 'USER' role in principal")
	}
}

func TestLoginWrongPasswordIncrementsFailures(t *testing.T) {
	svc, _, credRepo := newTestLocalAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "testuser", "test@example.com", "MyP@ssw0rd")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Login with wrong password.
	_, err = svc.Login(ctx, "testuser", "WrongPassword1")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}

	// Verify failed attempts count was incremented.
	cred := getCredByUsername(credRepo, "testuser")
	if cred == nil {
		t.Fatal("expected credential to exist")
	}
	if cred.FailedAttempts != 1 {
		t.Fatalf("expected 1 failed attempt, got %d", cred.FailedAttempts)
	}
}

func TestLoginLockedAccount(t *testing.T) {
	svc, _, _ := newTestLocalAuthService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "testuser", "test@example.com", "MyP@ssw0rd")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Fail login 5 times to trigger lock.
	for i := 0; i < 5; i++ {
		_, _ = svc.Login(ctx, "testuser", "WrongPassword1")
	}

	// 6th attempt should return locked error.
	_, err = svc.Login(ctx, "testuser", "MyP@ssw0rd")
	if err == nil {
		t.Fatal("expected error for locked account")
	}
}

func TestChangePasswordSucceeds(t *testing.T) {
	svc, userRepo, _ := newTestLocalAuthService()
	ctx := context.Background()

	result, err := svc.Register(ctx, "testuser", "test@example.com", "MyP@ssw0rd")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = svc.ChangePassword(ctx, result.User.ID, "MyP@ssw0rd", "NewP@ssw0rd!")
	if err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}

	// Should be able to login with new password.
	principal, err := svc.Login(ctx, "testuser", "NewP@ssw0rd!")
	if err != nil {
		t.Fatalf("Login with new password failed: %v", err)
	}
	if principal == nil {
		t.Fatal("expected non-nil principal after password change")
	}

	// Old password should no longer work.
	_, err = svc.Login(ctx, "testuser", "MyP@ssw0rd")
	if err == nil {
		t.Fatal("expected error when logging in with old password")
	}

	_ = userRepo // silence unused
}

func TestPlatformPrincipalHasRole(t *testing.T) {
	principal := PlatformPrincipal{
		UserID: "usr_abc",
		Roles:  []string{"ADMIN", "USER"},
	}

	if !principal.HasRole("USER") {
		t.Error("expected principal to have 'USER' role")
	}
	if !principal.HasRole("ADMIN") {
		t.Error("expected principal to have 'ADMIN' role")
	}
	if principal.HasRole("SUPER_ADMIN") {
		t.Error("expected principal to NOT have 'SUPER_ADMIN' role")
	}
}

func TestPlatformPrincipalIsSuperAdmin(t *testing.T) {
	adminPrincipal := PlatformPrincipal{
		UserID: "usr_admin",
		Roles:  []string{"SUPER_ADMIN", "USER"},
	}
	if !adminPrincipal.IsSuperAdmin() {
		t.Error("expected IsSuperAdmin to return true for SUPER_ADMIN role")
	}

	normalPrincipal := PlatformPrincipal{
		UserID: "usr_normal",
		Roles:  []string{"USER"},
	}
	if normalPrincipal.IsSuperAdmin() {
		t.Error("expected IsSuperAdmin to return false for non-SUPER_ADMIN")
	}
}

func TestNewPrincipalDefaultsUserRole(t *testing.T) {
	user := UserAccount{
		ID:          "usr_test",
		DisplayName: "TestUser",
		Email:       "test@example.com",
		Status:      "ACTIVE",
	}

	// No roles provided -> should default to USER.
	principal := NewPrincipal(user, "local", nil)
	if !principal.HasRole("USER") {
		t.Fatal("expected default 'USER' role when no roles assigned")
	}
	if len(principal.Roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(principal.Roles))
	}

	// With specific roles -> should use provided roles, not default.
	principal = NewPrincipal(user, "local", []string{"ADMIN"})
	if principal.HasRole("USER") {
		t.Error("expected no default 'USER' role when specific roles provided")
	}
	if !principal.HasRole("ADMIN") {
		t.Error("expected 'ADMIN' role from provided list")
	}
}

// ---- Helpers ----

func isSavedUser(repo *mockUserRepo, id string) bool {
	_, ok := repo.users[id]
	return ok
}

func isSavedCredential(repo *mockCredRepo, username string) bool {
	_, ok := repo.creds[username]
	return ok
}

func getCredByUsername(repo *mockCredRepo, username string) *LocalCredential {
	c, ok := repo.creds[username]
	if !ok {
		return nil
	}
	return &c
}
