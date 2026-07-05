// Package redis provides Redis-backed adapters for sessions and rate limiting.
package redis

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionStore provides Redis-backed session storage.
type SessionStore struct {
	client redis.UniversalClient
	ttl    time.Duration
}

// SessionConfig holds configuration for the session store.
type SessionConfig struct {
	URL string
	TTL time.Duration
}

// NewSessionStore creates a Redis-backed SessionStore.
func NewSessionStore(ctx context.Context, cfg SessionConfig) (*SessionStore, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("redis session: parse url: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis session: ping: %w", err)
	}
	return newSessionStoreWithClient(client, int(cfg.TTL.Seconds())), nil
}

// newSessionStoreWithClient creates a SessionStore with an existing client.
// Exported for tests.
func newSessionStoreWithClient(client redis.UniversalClient, ttlSeconds int) *SessionStore {
	return &SessionStore{
		client: client,
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

// redisClientFromAddr creates a redis client from an address string.
// Used by tests against miniredis.
func redisClientFromAddr(addr string) redis.UniversalClient {
	return redis.NewClient(&redis.Options{Addr: addr})
}

// Create generates a new session for the given user ID and returns the session ID.
func (s *SessionStore) Create(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("redis session: userID must not be empty")
	}

	// Generate 32 random bytes.
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("redis session: rand: %w", err)
	}

	// Encode as hex for the session ID (cookie value).
	sessionID := hex.EncodeToString(raw)

	// Store under hashed key.
	key := sessionKey(sessionID)
	if err := s.client.Set(ctx, key, userID, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("redis session: set: %w", err)
	}

	return sessionID, nil
}

// Validate checks if a session exists and returns the associated user ID.
func (s *SessionStore) Validate(ctx context.Context, sessionID string) (string, bool) {
	if sessionID == "" {
		return "", false
	}

	key := sessionKey(sessionID)
	userID, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return userID, true
}

// Delete removes a session from Redis.
func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	key := sessionKey(sessionID)
	// Redis DEL returns nil for missing keys, so this is safe to
	// return directly — the error is only non-nil on connection failures.
	return s.client.Del(ctx, key).Err()
}

// Ping checks connectivity to Redis.
func (s *SessionStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// Close releases the Redis client.
func (s *SessionStore) Close() error {
	return s.client.Close()
}

// IsAvailable returns true if the store is connected to Redis.
func (s *SessionStore) IsAvailable() bool {
	return s.client != nil
}

// sessionKey returns the Redis key for a session ID.
// The raw session ID is hashed so nobody with Redis access can read session
// tokens and replay them as cookies.
func sessionKey(sessionID string) string {
	hash := sha256.Sum256([]byte(sessionID))
	return "skillhub:session:" + hex.EncodeToString(hash[:])
}
