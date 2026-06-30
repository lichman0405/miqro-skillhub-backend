// Package redis provides Redis-backed adapters for sessions and other stateful services.
// In Phase 03, this is a placeholder. Full Redis integration is implemented in later phases.
package redis

import (
	"context"
	"fmt"
)

// SessionStore provides Redis-backed session storage.
// In Phase 03, this is a minimal stub that documents the interface.
type SessionStore struct {
	// Addr is the Redis server address.
	Addr string
}

// NewSessionStore creates a new SessionStore.
// In Phase 03, the store is not connected to a real Redis instance.
func NewSessionStore(addr string) *SessionStore {
	return &SessionStore{Addr: addr}
}

// Ping checks connectivity to Redis. Returns an error if Redis is not available.
func (s *SessionStore) Ping(ctx context.Context) error {
	// Placeholder: real implementation requires a Redis client (e.g., go-redis).
	// Phase 03 does not import a Redis driver.
	return fmt.Errorf("redis: not connected (Phase 03 placeholder)")
}

// IsAvailable returns true if the store is connected to a real Redis instance.
func (s *SessionStore) IsAvailable() bool {
	return false
}
