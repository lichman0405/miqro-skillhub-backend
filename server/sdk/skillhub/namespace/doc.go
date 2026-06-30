// Package namespace manages namespace lifecycle, members, ownership,
// and access policies.
//
// Source module mapping:
//
//	skillhub-domain domain/namespace
//	  Namespace entity with GLOBAL and TEAM types
//	  Lifecycle: ACTIVE, FROZEN, ARCHIVED
//	  NamespaceMember with OWNER, ADMIN, MEMBER roles
//	  NamespaceService for CRUD and lifecycle transitions
//	  Access policy enforcement outside HTTP handlers
//	  Namespace slug validation
//	  Member candidate resolution
//
// Implementation starts in Phase 04.
package namespace

