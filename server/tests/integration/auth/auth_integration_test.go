package auth_test

import (
	"context"
	"testing"

	"miqro-skillhub/server/internal/adapters/postgres"
	testpg "miqro-skillhub/server/internal/testutil/postgres"
	"miqro-skillhub/server/sdk/skillhub/auth"
)

func TestIntegrationAuth_SkipIfNoDB(t *testing.T) {
	db := testpg.TestDB(t)
	if db == nil {
		return // skipped
	}

	ctx := context.Background()
	repo := postgres.NewUserAccountRepo(db)

	// Create a test user.
	user := auth.UserAccount{
		ID:          "usr_integration_test",
		DisplayName: "Integration Test User",
		Email:       "integration@test.example.com",
		Status:      "ACTIVE",
	}
	saved, err := repo.Save(ctx, user)
	if err != nil {
		t.Fatalf("Save user failed: %v", err)
	}
	if saved.ID != "usr_integration_test" {
		t.Fatalf("expected user ID 'usr_integration_test', got %s", saved.ID)
	}

	// Retrieve the user.
	found, err := repo.FindByID(ctx, "usr_integration_test")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find user")
	}
	if found.DisplayName != "Integration Test User" {
		t.Fatalf("expected DisplayName 'Integration Test User', got %s", found.DisplayName)
	}
	if found.Email != "integration@test.example.com" {
		t.Fatalf("expected email 'integration@test.example.com', got %s", found.Email)
	}
	if found.Status != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %s", found.Status)
	}

	t.Log("Integration test passed against live PostgreSQL")
}
