package namespace_test

import (
	"context"
	"testing"

	"miqro-skillhub/server/internal/adapters/postgres"
	testpg "miqro-skillhub/server/internal/testutil/postgres"
	"miqro-skillhub/server/sdk/skillhub/namespace"
)

func TestIntegrationNamespace_SkipIfNoDB(t *testing.T) {
	db := testpg.TestDB(t)
	if db == nil {
		t.Skip("no database available")
		return
	}
	ctx := context.Background()

	nsRepo := postgres.NewNamespaceRepo(db)
	memRepo := postgres.NewNamespaceMemberRepo(db)

	// Create a namespace via the repo and verify slug uniqueness.
	ns := namespace.Namespace{
		Slug:        "integration-test",
		DisplayName: "Integration Test",
		Type:        "TEAM",
		Status:      "ACTIVE",
	}
	user := "test-user"

	created, err := nsRepo.Save(ctx, ns)
	if err != nil {
		t.Fatalf("Save namespace: %v", err)
	}
	t.Logf("Created namespace id=%d slug=%q", created.ID, created.Slug)

	// Slug uniqueness: inserting the same slug again should fail.
	ns2 := namespace.Namespace{
		Slug:        "integration-test",
		DisplayName: "Duplicate",
		Type:        "TEAM",
		Status:      "ACTIVE",
	}
	_, err = nsRepo.Save(ctx, ns2)
	if err != nil {
		t.Logf("Expected duplicate slug error: %v", err)
	} else {
		t.Error("expected error for duplicate slug in PostgreSQL")
	}

	// Create a member.
	member := namespace.NamespaceMember{
		NamespaceID: created.ID,
		UserID:      user,
		Role:        "OWNER",
	}
	savedMember, err := memRepo.Save(ctx, member)
	if err != nil {
		t.Fatalf("Save member: %v", err)
	}
	t.Logf("Created member id=%d role=%q", savedMember.ID, savedMember.Role)

	// Look up member.
	found, err := memRepo.FindByNamespaceAndUser(ctx, created.ID, user)
	if err != nil {
		t.Fatalf("FindByNamespaceAndUser: %v", err)
	}
	if found.Role != "OWNER" {
		t.Errorf("expected OWNER, got %q", found.Role)
	}

	// Find namespace by slug.
	fetched, err := nsRepo.FindBySlug(ctx, "integration-test")
	if err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if fetched.DisplayName != "Integration Test" {
		t.Errorf("expected DisplayName 'Integration Test', got %q", fetched.DisplayName)
	}

	t.Log("Integration test passed against live PostgreSQL")
}
