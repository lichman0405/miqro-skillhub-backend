package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func TestSessionStore_CreateValidateDelete(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	store := newSessionStoreWithClient(redisClientFromAddr(mr.Addr()), 3600)
	ctx := context.Background()

	sid, err := store.Create(ctx, "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sid == "" {
		t.Fatal("expected non-empty session ID")
	}

	userID, ok := store.Validate(ctx, sid)
	if !ok {
		t.Fatal("Validate: expected session to exist")
	}
	if userID != "user-1" {
		t.Fatalf("expected user-1, got %s", userID)
	}

	if err := store.Delete(ctx, sid); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, ok = store.Validate(ctx, sid)
	if ok {
		t.Fatal("Validate: expected session to be deleted")
	}
}

func TestSessionStore_ValidateMissingReturnsFalse(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	store := newSessionStoreWithClient(redisClientFromAddr(mr.Addr()), 3600)
	ctx := context.Background()

	_, ok := store.Validate(ctx, "nonexistent-session-id")
	if ok {
		t.Fatal("expected false for missing session")
	}
}

func TestSessionStore_KeysUseHashedSessionID(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	store := newSessionStoreWithClient(redisClientFromAddr(mr.Addr()), 3600)
	ctx := context.Background()

	sid, err := store.Create(ctx, "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// The raw session ID should NOT appear as a Redis key.
	keys := mr.Keys()
	for _, k := range keys {
		if k == sid {
			t.Fatalf("raw session ID %q must not be a Redis key", sid)
		}
	}

	// The hashed key should exist.
	hashedKey := sessionKey(sid)
	found := false
	for _, k := range keys {
		if k == hashedKey {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("hashed session key %q not found in Redis keys: %v", hashedKey, keys)
	}
}

func TestSessionStore_CreateRejectsEmptyUserID(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	store := newSessionStoreWithClient(redisClientFromAddr(mr.Addr()), 3600)
	ctx := context.Background()

	_, err = store.Create(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

func TestSessionStore_Ping(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	store := newSessionStoreWithClient(redisClientFromAddr(mr.Addr()), 3600)
	ctx := context.Background()

	if err := store.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestSessionStore_IsAvailable(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	store := newSessionStoreWithClient(redisClientFromAddr(mr.Addr()), 3600)
	if !store.IsAvailable() {
		t.Fatal("expected IsAvailable to return true")
	}
}

func TestSessionStore_SessionIDLength(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	store := newSessionStoreWithClient(redisClientFromAddr(mr.Addr()), 3600)
	ctx := context.Background()

	sid, err := store.Create(ctx, "user-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Session ID should be at least 32 raw bytes encoded (>= 43 base64 chars).
	if len(sid) < 43 {
		t.Fatalf("session ID too short: %d chars (expected >= 43)", len(sid))
	}
}
