// Package repository_test provides integration tests for repository implementations.
package repository_test

import (
	"context"
	"fmt"
	"testing"

	postgresadapters "miqro-skillhub/server/internal/adapters/postgres"
	testutil "miqro-skillhub/server/internal/testutil/postgres"
	"miqro-skillhub/server/sdk/skillhub/auth"
	"miqro-skillhub/server/sdk/skillhub/namespace"
)

func TestMigrationsApplyCleanly(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()

	tables := []string{
		"user_account", "identity_binding", "local_credential", "api_token",
		"role", "permission", "role_permission", "user_role_binding",
		"account_merge_request", "password_reset_request", "audit_log",
		"profile_change_request", "namespace", "namespace_member",
		"skill", "skill_version", "skill_file", "skill_tag",
		"skill_version_stats", "skill_storage_delete_compensation",
		"skill_search_document", "review_task", "promotion_request",
		"user_notification", "notification", "notification_preference",
		"idempotency_record", "label_definition", "label_translation",
		"skill_label", "skill_star", "skill_rating", "skill_subscription",
		"skill_report", "security_audit",
	}

	for _, table := range tables {
		var exists bool
		err := db.Pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = $1)`, table,
		).Scan(&exists)
		if err != nil {
			t.Errorf("Failed to check table %s: %v", table, err)
			continue
		}
		if !exists {
			t.Errorf("Table %s does not exist after migration", table)
		}
	}
}

func TestSeedDataApplied(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()

	var roleCount int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM role`).Scan(&roleCount); err != nil {
		t.Fatalf("Failed to count roles: %v", err)
	}
	if roleCount < 4 {
		t.Errorf("Expected at least 4 roles, got %d", roleCount)
	}

	var permCount int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM permission`).Scan(&permCount); err != nil {
		t.Fatalf("Failed to count permissions: %v", err)
	}
	if permCount < 8 {
		t.Errorf("Expected at least 8 permissions, got %d", permCount)
	}

	var rpCount int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM role_permission`).Scan(&rpCount); err != nil {
		t.Fatalf("Failed to count role_permission: %v", err)
	}
	if rpCount == 0 {
		t.Error("Expected role_permission bindings, got 0")
	}

	var nsSlug string
	if err := db.Pool.QueryRow(ctx, `SELECT slug FROM namespace WHERE slug = 'global'`).Scan(&nsSlug); err != nil {
		t.Errorf("Global namespace not found: %v", err)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()

	seedSQL := `
	INSERT INTO role (code, name, description, is_system) VALUES
		('SUPER_ADMIN', 'Super Administrator', 'Full platform access', TRUE)
	ON CONFLICT (code) DO NOTHING;

	INSERT INTO role (code, name, description, is_system) VALUES
		('SKILL_ADMIN', 'Skill Administrator', 'Skill review and management', TRUE)
	ON CONFLICT (code) DO NOTHING;
	`

	if _, err := db.Pool.Exec(ctx, seedSQL); err != nil {
		t.Errorf("Seed re-run failed: %v", err)
	}

	var roleCount int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM role`).Scan(&roleCount); err != nil {
		t.Fatalf("Failed to count roles: %v", err)
	}
	if roleCount != 4 {
		t.Errorf("Expected 4 roles after idempotent seed, got %d", roleCount)
	}
}

func TestUserAccountRepo_SaveAndFind(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()
	repo := postgresadapters.NewUserAccountRepo(db)

	user := auth.UserAccount{
		ID:          "test-user-1",
		DisplayName: "Test User",
		Email:       "test@example.com",
		Status:      "ACTIVE",
	}

	saved, err := repo.Save(ctx, user)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if saved.ID != "test-user-1" {
		t.Errorf("Expected ID 'test-user-1', got %q", saved.ID)
	}

	found, err := repo.FindByID(ctx, "test-user-1")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.DisplayName != "Test User" {
		t.Errorf("Expected 'Test User', got %q", found.DisplayName)
	}
	if found.Email != "test@example.com" {
		t.Errorf("Expected 'test@example.com', got %q", found.Email)
	}
}

func TestUserAccountRepo_FindByEmail(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()
	repo := postgresadapters.NewUserAccountRepo(db)

	user := auth.UserAccount{
		ID:          "test-user-2",
		DisplayName: "Email Test",
		Email:       "Email.Test@Example.com",
		Status:      "ACTIVE",
	}
	_, err := repo.Save(ctx, user)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	found, err := repo.FindByEmail(ctx, "email.test@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if found.ID != "test-user-2" {
		t.Errorf("Expected ID 'test-user-2', got %q", found.ID)
	}
}

func TestNamespaceRepo_SaveAndFind(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()
	repo := postgresadapters.NewNamespaceRepo(db)

	ns := namespace.Namespace{
		Slug:        "test-team",
		DisplayName: "Test Team",
		Type:        "TEAM",
		Status:      "ACTIVE",
	}

	saved, err := repo.Save(ctx, ns)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if saved.Slug != "test-team" {
		t.Errorf("Expected slug 'test-team', got %q", saved.Slug)
	}
	if saved.ID == 0 {
		t.Error("Expected non-zero ID after save")
	}

	found, err := repo.FindBySlug(ctx, "test-team")
	if err != nil {
		t.Fatalf("FindBySlug failed: %v", err)
	}
	if found.DisplayName != "Test Team" {
		t.Errorf("Expected 'Test Team', got %q", found.DisplayName)
	}
}

func TestNamespaceRepo_SlugUpsert(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()
	repo := postgresadapters.NewNamespaceRepo(db)

	ns1 := namespace.Namespace{Slug: "unique-slug", DisplayName: "First", Type: "TEAM", Status: "ACTIVE"}
	_, err := repo.Save(ctx, ns1)
	if err != nil {
		t.Fatalf("First save failed: %v", err)
	}

	ns2 := namespace.Namespace{Slug: "unique-slug", DisplayName: "Second", Type: "TEAM", Status: "ACTIVE"}
	saved, err := repo.Save(ctx, ns2)
	if err != nil {
		t.Errorf("Upsert on duplicate slug failed: %v", err)
	}
	if saved.DisplayName != "Second" {
		t.Errorf("Expected display name 'Second' after upsert, got %q", saved.DisplayName)
	}
}

func TestNamespaceMemberRepo_CRUD(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()

	userRepo := postgresadapters.NewUserAccountRepo(db)
	_, err := userRepo.Save(ctx, auth.UserAccount{ID: "member-user-1", DisplayName: "Member User", Status: "ACTIVE"})
	if err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	ns := namespace.Namespace{Slug: "member-test", DisplayName: "Member Test", Type: "TEAM", Status: "ACTIVE"}
	nsRepo := postgresadapters.NewNamespaceRepo(db)
	savedNS, err := nsRepo.Save(ctx, ns)
	if err != nil {
		t.Fatalf("Create namespace failed: %v", err)
	}

	memberRepo := postgresadapters.NewNamespaceMemberRepo(db)
	member := namespace.NamespaceMember{
		NamespaceID: savedNS.ID,
		UserID:      "member-user-1",
		Role:        "OWNER",
	}

	saved, err := memberRepo.Save(ctx, member)
	if err != nil {
		t.Fatalf("Save member failed: %v", err)
	}
	if saved.ID == 0 {
		t.Error("Expected non-zero ID after save")
	}

	found, err := memberRepo.FindByNamespaceAndUser(ctx, savedNS.ID, "member-user-1")
	if err != nil {
		t.Fatalf("FindByNamespaceAndUser failed: %v", err)
	}
	if found.Role != "OWNER" {
		t.Errorf("Expected role 'OWNER', got %q", found.Role)
	}
}

func TestTransactor_RollbackOnError(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()
	transactor := postgresadapters.NewTransactor(db.Pool)
	userRepo := postgresadapters.NewUserAccountRepo(db)

	err := transactor.WithinTx(ctx, func(txCtx context.Context) error {
		_, saveErr := userRepo.Save(txCtx, auth.UserAccount{
			ID:          "rollback-user",
			DisplayName: "Rollback Test",
			Status:      "ACTIVE",
		})
		if saveErr != nil {
			return saveErr
		}
		return fmt.Errorf("intentional rollback")
	})

	if err == nil {
		t.Error("Expected error from WithinTx, got nil")
	}

	// Verify the user was NOT persisted (rolled back).
	_, err = userRepo.FindByID(ctx, "rollback-user")
	if err == nil {
		t.Error("Expected user to be rolled back, but it was found")
	}
}

func TestTransactor_CommitOnSuccess(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()
	transactor := postgresadapters.NewTransactor(db.Pool)
	userRepo := postgresadapters.NewUserAccountRepo(db)

	err := transactor.WithinTx(ctx, func(txCtx context.Context) error {
		_, saveErr := userRepo.Save(txCtx, auth.UserAccount{
			ID:          "commit-user",
			DisplayName: "Commit Test",
			Status:      "ACTIVE",
		})
		return saveErr
	})

	if err != nil {
		t.Fatalf("WithinTx failed: %v", err)
	}

	found, err := userRepo.FindByID(ctx, "commit-user")
	if err != nil {
		t.Fatalf("Expected user to be committed, but FindByID failed: %v", err)
	}
	if found.DisplayName != "Commit Test" {
		t.Errorf("Expected 'Commit Test', got %q", found.DisplayName)
	}
}

func TestMigrateUp_IsIdempotent(t *testing.T) {
	db := testutil.TestDB(t)
	if db == nil {
		return
	}

	ctx := context.Background()

	// First run.
	if err := postgresadapters.MigrateUp(ctx, db.Pool); err != nil {
		t.Fatalf("First MigrateUp failed: %v", err)
	}

	// Second run should succeed (all migrations already applied).
	if err := postgresadapters.MigrateUp(ctx, db.Pool); err != nil {
		t.Fatalf("Second MigrateUp (idempotency check) failed: %v", err)
	}

	var roleCount int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM role`).Scan(&roleCount); err != nil {
		t.Fatalf("Failed to count roles: %v", err)
	}
	if roleCount != 4 {
		t.Errorf("Expected 4 roles, got %d (seed was duplicated)", roleCount)
	}
}
