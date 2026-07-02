# API Usage Guide

## API base URL

```
http://localhost:8080
```

All API routes are under `/api/v1/` (portal), `/api/tool/v1/` (CLI tooling), or `/api/cli/v1/` (CLI search).

## Browser CORS

Browser clients running on a different origin must be explicitly allowlisted:

```bash
SKILLHUB_CORS_ALLOWED_ORIGINS=http://localhost:5173,https://app.example.com
```

Leave the value empty for same-origin only. `*` is allowed only for non-credentialed local experiments; credentialed requests require explicit origins.

## Authentication

SkillHub supports two auth methods:

### Bearer token

```
Authorization: Bearer sk_...
```

Create a token via `POST /api/v1/auth/tokens` after logging in.

### Session cookie

Login via `POST /api/v1/auth/login` — the server sets a `skillhub_session` cookie.

### Local mode

When `SKILLHUB_LOCAL_MODE=true` (default), the server auto-creates a local admin user `admin` / `admin`. API tokens are accepted with permissive auth.

## Response envelope

All responses use a standard envelope:

```json
{
  "success": true,
  "data": { ... },
  "error": { "code": "not_found", "message": "skill not found" }
}
```

On success, `success` is `true` and `data` contains the response payload.

On error, `success` is `false`, `data` is absent, and `error` contains:

| Field | Description |
|---|---|
| `code` | Machine-readable error code (`bad_request`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `internal`) |
| `message` | Human-readable error message |

### HTTP status codes

| Status | Meaning |
|---|---|
| `200` | Success |
| `201` | Created |
| `400` | Bad request — invalid input |
| `401` | Unauthorized — missing or invalid credentials |
| `403` | Forbidden — authenticated but not allowed |
| `404` | Not found |
| `409` | Conflict — duplicate resource or gate enforcement failed |
| `500` | Internal server error |
| `503` | Service unavailable — backend not configured |

## Pagination

List endpoints support pagination via query parameters:

| Parameter | Default | Max | Description |
|---|---|---|---|
| `page` | `0` | — | Zero-based page index |
| `size` | `20` | `100` | Items per page |

Response includes:

```json
{
  "totalCount": 142,
  "page": 0,
  "size": 20
}
```

## Sorting / filter conventions

Search endpoints accept:

| Parameter | Values |
|---|---|
| `sortBy` | `relevance`, `downloads`, `rating`, `newest` |
| `keyword` | Free-text search string |

Community list endpoints accept `status` filter (e.g., `?status=OPEN`) and `category` filter (e.g., `?category=QA`).

---

## Auth

### Login

```
POST /api/v1/auth/login
Content-Type: application/json

{"username": "admin", "password": "admin"}
```

Response:
```json
{
  "success": true,
  "data": {
    "userID": "admin",
    "displayName": "Admin",
    "email": "",
    "authMethod": "local",
    "platformRoles": {"SUPER_ADMIN": true},
    "isAuthenticated": true
  }
}
```

### Register

```
POST /api/v1/auth/register
Content-Type: application/json

{"username": "newuser", "password": "secret123", "displayName": "New User", "email": "user@example.com"}
```

### Get current user

```
GET /api/v1/auth/me
Authorization: Bearer sk_...
```

### Logout

```
POST /api/v1/auth/logout
Authorization: Bearer sk_...
```

### API tokens

```
GET /api/v1/auth/tokens          # list tokens
POST /api/v1/auth/tokens          # create token
DELETE /api/v1/auth/tokens/{id}   # revoke token
```

### Password reset

```
POST /api/v1/auth/password-reset/request   # request reset email
POST /api/v1/auth/password-reset/confirm   # confirm with token from email
```

---

## Search

```
GET /api/v1/search?keyword=agent&sortBy=downloads&installableOnly=true
POST /api/v1/search
Content-Type: application/json

{"keyword": "agent", "installableOnly": true}
```

Response:
```json
{
  "success": true,
  "data": {
    "skillIds": [1, 5, 12],
    "total": 3,
    "page": 1,
    "size": 20
  }
}
```

---

## Namespace

### List namespaces

```
GET /api/v1/namespaces
```

### Get namespace

```
GET /api/v1/namespaces/{slug}
```

Response:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "slug": "global",
    "displayName": "Global",
    "type": "GLOBAL",
    "description": "Global namespace"
  }
}
```

### Create namespace
```
POST /api/v1/namespaces
Authorization: Bearer sk_...
{"slug": "my-org", "displayName": "My Org", "type": "PUBLIC"}
```

### Manage members
```
GET /api/v1/namespaces/{id}/members
POST /api/v1/namespaces/{id}/members          # add member
DELETE /api/v1/namespaces/{id}/members/{userID}  # remove member
```

---

## Skill

### Get skill detail

```
GET /api/v1/skills/{namespace}/{slug}
```

Response:
```json
{
  "success": true,
  "data": {
    "id": 1,
    "slug": "my-skill",
    "displayName": "My Skill",
    "ownerId": "admin",
    "summary": "A useful agent skill",
    "visibility": "PUBLIC",
    "status": "ACTIVE",
    "downloadCount": 42,
    "starCount": 5,
    "ratingAvg": 4.5,
    "canManage": true
  }
}
```

### List versions

```
GET /api/v1/skills/{namespace}/{slug}/versions
```

### Get version detail

```
GET /api/v1/skills/{namespace}/{slug}/versions/{version}
```

### List files

```
GET /api/v1/skills/{namespace}/{slug}/files
```

### Download

```
GET /api/v1/skills/{namespace}/{slug}/download
```

Returns a ZIP archive of the latest published version.

### Publish

```
POST /api/v1/skills/{namespace}/publish
Authorization: Bearer sk_...
Content-Type: multipart/form-data

package: <zip file>
```

Response:
```json
{
  "success": true,
  "data": {
    "skillId": 1,
    "slug": "my-skill",
    "version": {"id": 10, "version": "1.0.0", "status": "PUBLISHED"}
  }
}
```

---

## Release

### List releases

```
GET /api/v1/skills/{namespace}/{slug}/releases?page=0&size=20
```

### Get latest stable release

```
GET /api/v1/skills/{namespace}/{slug}/releases/latest
GET /api/v1/skills/{namespace}/{slug}/releases/latest?channel=beta
```

### Get release by ID

```
GET /api/v1/skills/{namespace}/{slug}/releases/{releaseID}
```

Response includes the release and its assets:
```json
{
  "release": {
    "id": 1,
    "skillId": 1,
    "versionId": 5,
    "channel": "stable",
    "title": "v1.0.0",
    "notes": "Initial release",
    "draft": true,
    "prerelease": false,
    "yanked": false,
    "publishedAt": null,
    "publisherId": "admin"
  },
  "assets": []
}
```

### Create release

```
POST /api/v1/skills/{namespace}/{slug}/releases
Authorization: Bearer sk_...
Content-Type: application/json

{
  "versionId": 5,
  "channel": "stable",
  "title": "v1.0.0",
  "notes": "Initial stable release"
}
```

All releases are created as **drafts**. Call `publish` to make them live.

### Update release

```
PATCH /api/v1/skills/{namespace}/{slug}/releases/{releaseID}
Authorization: Bearer sk_...
{"title": "Updated title", "notes": "Updated notes"}
```

Note: `draft: false` is rejected via PATCH — use the publish endpoint instead.

### Delete release

```
DELETE /api/v1/skills/{namespace}/{slug}/releases/{releaseID}
Authorization: Bearer sk_...
```

### Publish release

```
POST /api/v1/skills/{namespace}/{slug}/releases/{releaseID}/publish
Authorization: Bearer sk_...
```

This runs CI gate enforcement before publishing. Returns `409 Conflict` if gates are not satisfied.

---

## Community

### Issues

```
GET    /api/v1/skills/{namespace}/{slug}/issues                    # list
POST   /api/v1/skills/{namespace}/{slug}/issues                    # create
GET    /api/v1/skills/{namespace}/{slug}/issues/{issueID}           # get
PATCH  /api/v1/skills/{namespace}/{slug}/issues/{issueID}           # update
DELETE /api/v1/skills/{namespace}/{slug}/issues/{issueID}           # delete
GET    /api/v1/skills/{namespace}/{slug}/issues/{issueID}/comments  # list comments
POST   /api/v1/skills/{namespace}/{slug}/issues/{issueID}/comments  # add comment
```

Create issue example:
```json
{
  "title": "Bug: SKILL.md parser fails on Windows paths",
  "body": "When uploading a package with backslash paths..."
}
```

### Discussions

```
GET    /api/v1/skills/{namespace}/{slug}/discussions
POST   /api/v1/skills/{namespace}/{slug}/discussions
GET    /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}
PATCH  /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}
DELETE /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}
GET    /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}/comments
POST   /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}/comments
POST   /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}/accept-answer
```

### Wiki pages

```
GET    /api/v1/skills/{namespace}/{slug}/wiki
POST   /api/v1/skills/{namespace}/{slug}/wiki
GET    /api/v1/skills/{namespace}/{slug}/wiki/{pageSlug}
PUT    /api/v1/skills/{namespace}/{slug}/wiki/{pageSlug}
DELETE /api/v1/skills/{namespace}/{slug}/wiki/{pageSlug}
GET    /api/v1/skills/{namespace}/{slug}/wiki/{pageSlug}/versions
```

### Change proposals

```
GET    /api/v1/skills/{namespace}/{slug}/proposals
POST   /api/v1/skills/{namespace}/{slug}/proposals
GET    /api/v1/skills/{namespace}/{slug}/proposals/{proposalID}
PATCH  /api/v1/skills/{namespace}/{slug}/proposals/{proposalID}
```

### Community search

```
GET /api/v1/skills/{namespace}/{slug}/community/search?query=bug&types=issues,discussions
```

---

## Agent CI

### Pipeline runs

```
GET /api/v1/skills/{skillID}/ci/runs?page=0&size=20
GET /api/v1/skills/{skillID}/ci/runs/{runID}
```

Response:
```json
{
  "id": 1,
  "pipelineId": 1,
  "skillId": 1,
  "versionId": 5,
  "triggerType": "publish",
  "triggeredBy": "admin",
  "status": "COMPLETED",
  "checkCount": 3,
  "passedCount": 2,
  "failedCount": 0,
  "skippedCount": 1,
  "startedAt": "2026-01-01T00:00:00Z",
  "completedAt": "2026-01-01T00:01:00Z"
}
```

### Check runs

```
GET /api/v1/skills/{skillID}/ci/runs/{runID}/checks
GET /api/v1/skills/{skillID}/ci/checks/{checkID}
GET /api/v1/skills/{skillID}/ci/checks/{checkID}/artifacts
```

### Gate evaluation

```
GET /api/v1/skills/{skillID}/ci/gates?trigger=publish&versionId=5
```

Response:
```json
{
  "passed": true,
  "policyResults": [
    {"policyId": 1, "policyName": "manifest-validation", "passed": true}
  ]
}
```

---

## Frontend read models

Frontend routes provide viewer-scoped read models with `availableActions` computed from the authenticated user's permissions.

Each route returns a data object containing the page data plus an `availableActions` object with boolean flags the frontend uses to show/hide UI elements.

Implementation status:

| Route group | Data status |
|---|---|
| Search/home | Real SDK search result IDs, pagination, labels/installable filters, viewer visibility scope |
| Skill detail/version detail | Real SDK skill and version detail when the skill service is wired |
| Namespace list/detail | Real ACTIVE namespace list and authorized member list |
| Release list/detail | Real release and asset data scoped to the requested skill |
| Issues/discussions/wiki/proposals | Real community read models from Phase 11 |
| Reviews/promotions | Real queue and detail read models with skill/version/namespace enrichment; row-level action flags |
| Governance | Real notification summary/activity plus pending review/promotion counts scoped to viewer permissions |
| Admin | Real aggregate stats for SUPER_ADMIN; unauthorized viewers receive zero stats |

```
GET /api/v1/frontend/search?q=agent&page=0&size=20&sort=downloads&labels=go,agent&installable=true
GET /api/v1/frontend/skills/{namespace}/{slug}            # skill detail
GET /api/v1/frontend/skills/{namespace}/{slug}/versions/{version}  # version detail
GET /api/v1/frontend/skills/{namespace}/publish/validate  # publish page
GET /api/v1/frontend/namespaces                           # namespace list
GET /api/v1/frontend/namespaces/{slug}                    # namespace detail
GET /api/v1/frontend/reviews                              # review queue
GET /api/v1/frontend/reviews/{id}                         # review detail
GET /api/v1/frontend/promotions                           # promotion queue
GET /api/v1/frontend/promotions/{id}                      # promotion detail
GET /api/v1/frontend/governance                           # governance workbench
GET /api/v1/frontend/admin                                # admin dashboard
GET /api/v1/frontend/skills/{namespace}/{slug}/releases   # release list
GET /api/v1/frontend/skills/{namespace}/{slug}/releases/{releaseID}  # release detail
GET /api/v1/frontend/skills/{namespace}/{slug}/issues     # issue list
GET /api/v1/frontend/skills/{namespace}/{slug}/issues/{issueID}      # issue detail
GET /api/v1/frontend/skills/{namespace}/{slug}/discussions           # discussion list
GET /api/v1/frontend/skills/{namespace}/{slug}/discussions/{discussionID}  # discussion detail
GET /api/v1/frontend/skills/{namespace}/{slug}/wiki                  # wiki list
GET /api/v1/frontend/skills/{namespace}/{slug}/wiki/{pageSlug}       # wiki detail
GET /api/v1/frontend/skills/{namespace}/{slug}/proposals             # proposal list
GET /api/v1/frontend/skills/{namespace}/{slug}/proposals/{proposalID} # proposal detail
```

---

## Tool API (miqro CLI)

```
GET  /api/tool/v1/workspace/metadata                       # init contract
POST /api/tool/v1/packages/hash                            # compute package hash
GET  /api/tool/v1/skills/{namespace}/{slug}/resolve         # resolve version
GET  /api/tool/v1/skills/{namespace}/{slug}/install         # install metadata
GET  /api/tool/v1/skills/{namespace}/{slug}/diff?from=v1&to=v2  # version diff
POST /api/tool/v1/skills/{namespace}/validate               # dry-run validation (multipart)
POST /api/tool/v1/skills/{namespace}/publish                # publish package (multipart)
POST /api/tool/v1/evaluate/trigger                          # evaluate trigger (Phase 12 placeholder)
POST /api/tool/v1/proposals/prepare                         # proposal prepare (Phase 11 placeholder)
```

---

## Other endpoints

```
GET /healthz                          # liveness probe
GET /readyz                           # readiness probe
GET /.well-known/skillhub             # registry discovery
GET /.well-known/clawhub              # ClawHub compatibility
GET /metrics                          # Prometheus metrics (text/plain)
```

## CLI API (`/api/cli/v1`)

Legacy CLI interface for the miqro CLI tool:

```
GET  /api/cli/v1/auth/whoami                                        # CLI whoami
GET  /api/cli/v1/skills/search?q=...                                # CLI search
GET  /api/cli/v1/skills/{namespace}/{slug}/resolve                  # CLI resolve
GET  /api/cli/v1/skills/{namespace}/{slug}/download                 # CLI download latest
GET  /api/cli/v1/skills/{namespace}/{slug}/versions/{version}/download  # CLI download specific version
POST /api/cli/v1/skills/{namespace}/publish/validate                # CLI dry-run validate (multipart)
POST /api/cli/v1/skills/{namespace}/publish                         # CLI publish (multipart)
DELETE /api/cli/v1/skills/{namespace}/{slug}                        # CLI delete skill
```
