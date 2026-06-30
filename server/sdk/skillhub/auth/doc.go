// Package auth provides local authentication, direct auth, sessions,
// device auth, API tokens, scopes, and RBAC principal resolution.
//
// Source module mapping:
//
//	skillhub-auth (server/skillhub-auth)
//	  OAuth / OIDC login, GitHub/GitLab claims, route resolver
//	  Local auth, password policy, password reset
//	  Direct auth provider (username/password)
//	  Device auth for CLI
//	  Passive session authenticator, Redis sessions
//	  API token service, scope filter, token hashing (SHA-256)
//	  RBAC: PlatformPrincipal, role defaults, role bindings
//	  Account merge service
//
// Tokens:
//   - Raw token starts with "sk_"
//   - 32 random bytes, base64url encoded
//   - Only SHA-256 hash stored; prefix = first 8 chars
//   - Active token names are unique per user (case-insensitive)
//   - Expiration accepts Instant, OffsetDateTime, or legacy naive UTC
//   - Revocation is idempotent for missing/foreign token IDs
//   - Scope JSON is parsed as a list of strings
//
// Implementation starts in Phase 03.
package auth

// Service is a placeholder that will hold auth domain logic starting in Phase 03.
type Service struct{}

