# End-to-End Flow

The complete happy path from skill package upload to release publication.

## Overview

```
Package upload → Validate → Publish version → CI pipeline runs → Review approve → Create release → Publish release → Download/install
```

## Step-by-step

### 1. Namespace setup

A namespace is required before publishing skills.

**Action:** Create a namespace (or join the global namespace).

```
POST /api/v1/namespaces
{"slug": "my-org", "displayName": "My Org", "type": "PUBLIC"}
```

The `global` namespace exists by default (seed data).

**Status:** ✅ Implemented

### 2. Skill package upload (CLI: miqro publish)

The user packages their skill files (SKILL.md + optional files) into a zip, then publishes.

```
POST /api/tool/v1/skills/{namespace}/publish
Content-Type: multipart/form-data
package: <skill.zip>
```

The server:
1. Validates the zip structure (SKILL.md required)
2. Extracts files, computes SHA-256 hashes
3. Stores files in object storage (local filesystem or S3/MinIO)
4. Creates or updates the skill record
5. Creates a new version

**Status:** ✅ Implemented (zip extraction, file storage, version creation)

### 3. Package validation (CLI: miqro validate)

Dry-run validation before publishing:

```
POST /api/tool/v1/skills/{namespace}/validate
Content-Type: multipart/form-data
package: <skill.zip>
```

**Status:** ✅ Implemented

### 4. Version published

After a successful publish, the version is in `PUBLISHED` status. The version record includes:
- Version string (`1.0.0`)
- List of package files with SHA-256 hashes and storage keys
- Manifest metadata from SKILL.md

**Status:** ✅ Implemented

### 5. CI pipeline trigger

When a version is published, the server creates a CI pipeline run with all configured checks:

- **manifest-validation** — deterministic `SKILL.md` manifest check
- **secret-scan** — scans files for secrets/key patterns
- **documentation-quality** — checks documentation completeness
- **llm-review** — LLM-powered quality review (optional, requires `AGENTCI_LLM_*` env vars)

**Status:** ✅ Implemented (pipeline creation and trigger). Local deterministic runner is wired; LLM runner requires configuration.

**Known gap:** LLM runner returns SKIPPED without `AGENTCI_LLM_*` env vars. This is by design — the deterministic checks still run.

### 6. Worker executes CI checks

The `skillhub-worker` polls for PENDING pipeline runs, claims them (atomic PENDING→RUNNING), and executes all check runs.

```
skillhub-worker: found 1 pending runs
skillhub-worker: executing pipeline run 1 (skill=1, checks=3)
```

**Status:** ✅ Implemented. Worker polls every 30s (configurable via `AGENTCI_POLL_INTERVAL`). Uses `ClaimPendingRun` for concurrency safety.

### 7. CI gate evaluation

Before publishing a release or approving a review, the system evaluates CI gates:

```
GET /api/v1/skills/{skillID}/ci/gates?trigger=release_publish&versionId=5
```

Response:
```json
{
  "passed": true,
  "policyResults": [
    {"policyId": 1, "policyName": "manifest-validation", "passed": true},
    {"policyId": 2, "policyName": "secret-scan", "passed": true},
    {"policyId": 3, "policyName": "documentation-quality", "passed": true}
  ]
}
```

**Status:** ✅ Implemented. Gates are evaluated per policy; blocking policies that fail prevent publishing.

### 8. Review and approval (optional)

For namespaces that require review, the version must go through review before release:

1. Skill owner submits for review
2. Reviewer sees the review in the review queue
3. Reviewer approves (or rejects)

```
POST /api/v1/frontend/reviews/{id}/approve
```

**Gate enforcement during review:** When a reviewer approves, the `review_approve` trigger evaluates CI gates. If gates fail, approval is blocked (409).

**Status:** ✅ Implemented (SDK-level gate enforcement). Review service has `GateEnforcer` wired in `main.go`.

**Known gap:** No HTTP review approval handler exists yet. Review submission (`POST`) is implemented but the approve/reject endpoints are not yet exposed as HTTP routes. The SDK-level `ApproveReview` with gate enforcement is implemented and tested.

### 9. Create release

Create a release from a published version:

```
POST /api/v1/skills/{namespace}/{slug}/releases
{"versionId": 5, "channel": "stable", "title": "v1.0.0", "notes": "Initial release"}
```

**All releases are created as drafts** (`draft: true`). The `draft` field in the request body is ignored — the server always forces draft.

**Status:** ✅ Implemented. `CreateRelease` forces `draft=true`.

### 10. Publish release (with gate enforcement)

To make a release live, call the publish endpoint:

```
POST /api/v1/skills/{namespace}/{slug}/releases/{releaseID}/publish
```

The server:
1. Verifies the caller is the publisher or super admin
2. Runs CI gate enforcement (`release_publish` trigger)
3. If gates pass → sets `draft=false`, records `publishedAt`
4. If gates fail → returns 409 with reason

**Status:** ✅ Implemented.

**Gate bypass prevention:** `UpdateRelease` (PATCH) rejects `draft: false` — the only path from draft to published is through `PublishRelease`.

### 11. Download / install

Once a release is published, users can download and install:

```
GET /api/v1/skills/{namespace}/{slug}/download     # download latest zip
GET /api/tool/v1/skills/{namespace}/{slug}/install  # install metadata
GET /api/tool/v1/skills/{namespace}/{slug}/resolve  # resolve version + fingerprint
```

**Status:** ✅ Implemented (download, resolve). Install metadata returns target info including supported agent runtimes.

## Flow diagram

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  miqro CLI  │────▶│  /tool/v1/   │────▶│  PostgreSQL  │
│  (publish)  │     │  publish     │     │  (version)   │
└─────────────┘     └──────┬───────┘     └──────┬───────┘
                           │                    │
                    ┌──────▼───────┐     ┌──────▼───────┐
                    │  CI Pipeline │     │ Object Store  │
                    │  Trigger     │     │ (package zip) │
                    └──────┬───────┘     └──────────────┘
                           │
                    ┌──────▼───────┐
                    │   Worker     │
                    │  (poll +     │
                    │   execute)   │
                    └──────┬───────┘
                           │
              ┌────────────▼────────────┐
              │  CI Checks Complete     │
              │  (manifest, secrets,    │
              │   docs, llm-review)     │
              └────────────┬────────────┘
                           │
              ┌────────────▼────────────┐
              │  Review (optional)      │
              │  Gate: review_approve   │
              └────────────┬────────────┘
                           │
              ┌────────────▼────────────┐
              │  Create Release (draft) │
              └────────────┬────────────┘
                           │
              ┌────────────▼────────────┐
              │  Publish Release        │
              │  Gate: release_publish  │
              └────────────┬────────────┘
                           │
              ┌────────────▼────────────┐
              │  Download / Install     │
              └─────────────────────────┘
```

## Status summary

| Step | Status | Notes |
|---|---|---|
| Namespace setup | ✅ Implemented | Global namespace seeded; CRUD via API |
| Package upload (publish) | ✅ Implemented | Zip extraction, file storage, version creation |
| Package validation | ✅ Implemented | Dry-run via tool API |
| CI pipeline creation | ✅ Implemented | Checks created on publish trigger |
| Worker execution | ✅ Implemented | Poll + ClaimPending + execute |
| Deterministic checks | ✅ Implemented | manifest-validation, secret-scan, documentation-quality |
| LLM-powered checks | 🔶 Stub | Returns SKIPPED without LLM config |
| Step logs | 🔶 Not wired | `LogStore` remains nil — step logs not persisted |
| CI gate evaluation | ✅ Implemented | Per-policy evaluation with pass/fail |
| Review submission | ✅ Implemented | SDK complete |
| Review approval HTTP | 🔶 Missing | SDK-level `ApproveReview` with gate works; HTTP handler not exposed |
| Create release (draft) | ✅ Implemented | Always draft; gate bypass blocked |
| Publish release | ✅ Implemented | Gate enforcement at publish |
| Release download | ✅ Implemented | ZIP download of latest published |
| Install metadata | ✅ Implemented | Tool API returns target info |
| S3/MinIO storage | 🔶 Not wired | Local filesystem only; MinIO adapter exists but not configured for agent CI |
