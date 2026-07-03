# Frontend Integration Guide

How to call the SkillHub backend from each frontend page. Every page uses a **frontend read-model** endpoint that returns the page data plus `availableActions` — a set of boolean flags computed from the authenticated user's permissions.

## General patterns

### Local browser setup

When the frontend runs on a different origin, configure the backend before starting `skillhub-server`:

```bash
SKILLHUB_CORS_ALLOWED_ORIGINS=http://localhost:5173
```

Use explicit origins for authenticated browser requests. Do not rely on wildcard CORS for session cookies or bearer-token based UI traffic.

### Contract fixtures

Representative frontend read-model response fixtures live under:

```text
server/internal/http/frontend/testdata/contracts/
```

Use these fixtures to design frontend loading states, TypeScript mocks, and Storybook/demo data. They are examples of current response envelopes, not a replacement for API tests. Each fixture is a full `{ success: true, data: ... }` envelope covering one of the ready frontend read-model routes.

### Read-model implementation status

| Area | Current status |
|---|---|
| Search/home | Real backend search IDs and pagination, scoped by viewer visibility |
| Skill detail/version detail | Real SDK data when the backend services are wired |
| Namespace list/detail | Real ACTIVE namespace list and authorized member list |
| Release list/detail | Real release and asset data scoped to the requested skill |
| Community | Real issue/discussion/wiki/proposal read models |
| Review queue/detail | Real review task rows with skill/version/namespace enrichment |
| Promotion queue/detail | Real promotion request rows with source skill/version and target namespace enrichment |
| Governance workbench | Real notification summary/activity plus pending review/promotion counts for authorized roles |
| Admin dashboard | Real aggregate stats for SUPER_ADMIN; zero stats for unauthorized viewers |

### Use the TypeScript SDK

Prefer the `@miqro/skillhub-client` TypeScript SDK over raw `fetch()` calls. It handles auth, URL encoding, error normalization, and pagination:

```typescript
import { SkillHubClient, SkillHubError } from "@miqro/skillhub-client";

const client = new SkillHubClient({
  baseUrl: "http://localhost:8080",
  credentials: "include", // cookie-based session auth
});

// Envelope mode (compatible)
const result = await client.frontendSearch({ keyword: "agent" });
if (result.success && result.data) {
  // use result.data
}

// Unwrap mode (throwing)
try {
  const data = await client.unwrap(client.frontendSkillDetail("ns", "slug"));
  console.log(data.availableActions.canEdit);
} catch (err) {
  if (err instanceof SkillHubError) {
    // handle typed error
  }
}
```

### Loading flow for every page

1. **Check auth** — call `client.me()`. If 401, show login.
2. **Fetch read model** — call the page's frontend SDK method.
3. **Render** — use data for content, `availableActions` to show/hide controls.
4. **Mutations** — call the corresponding SDK portal method, then re-fetch the read model.

### Button visibility

**Never hard-code role checks on the frontend.** Use `availableActions` from the read model:

```typescript
const { data } = await client.frontendSkillDetail(ns, slug);
if (data.availableActions.canEdit) {
  // show edit button
}
```

### Empty states

| Scenario | Frontend behavior |
|---|---|
| No search results | Show "No skills found" with a "Create your first skill" CTA (if `canCreateSkill`) |
| No versions | Show "No versions published yet" |
| No releases | Show "No releases yet" with "Create release" button (if `canCreateRelease`) |
| No issues/discussions | Show "Be the first to start a discussion" with create button (if permission allows) |
| No reviews in queue | Show "All caught up! No pending reviews." |
| No CI runs | Show "No CI pipeline runs yet. Publish a version to trigger CI." |

### Error states

- **503 Service Unavailable** — backend is starting or database is unreachable. Show a "Server starting..." message with retry.
- **Network error** — Show "Cannot reach server. Check your connection." with retry button.
- **401 Unauthorized** — Redirect to login.
- **403 Forbidden** — Show "You don't have permission to do that."
- **404 Not Found** — Show "Not found" page.
- **409 Conflict** — Show the conflict reason (e.g., "Gate enforcement failed: manifest-validation did not pass").

### Pagination

Use the SDK's bounded async iterators for user-driven paging or bounded background prefetch. Every iterator defaults to `maxPages: 10` and stops when no more data is available:

```typescript
// User-driven review queue browsing
for await (const page of client.iterFrontendReviews({ size: 50, maxPages: 5 })) {
  for (const task of page.tasks) {
    renderTask(task);
  }
}

// Bounded community issue prefetch
for await (const page of client.iterFrontendIssues("ns", "skill", { maxPages: 3 })) {
  cacheIssues(page.issues);
}
```

Do not use iterators without `maxPages` for unbounded background prefetch. For manual paging, call the underlying SDK method directly with `page` and `size` parameters.

### Review/promotion mutation endpoints

Review and promotion frontend endpoints are read-model only. There are currently no HTTP endpoints to approve, reject, or withdraw reviews or promotions. If the frontend needs mutation buttons, implement separate HTTP routes backed by the SDK `review` and `promotion` services. Do not assume `POST /api/v1/frontend/reviews/{id}/approve` or similar routes exist.

---

## Discover page (Home/Search)

**Endpoint:** `GET /api/v1/frontend/search`

**Data returned:**
- `searchResult` — skill IDs, total count, pagination
- `featuredLabels` — trending/popular labels
- `availableActions` — `canCreateSkill`, `canCreateNamespace`, `canAccessAdmin`

**Recommended loading order:**
1. Call `GET /api/v1/auth/me` to check auth
2. Call `GET /api/v1/frontend/search?q=...&page=0&size=20&sort=relevance`
3. For installable-only discovery, add `installable=true`; for labels, pass `labels=go,agent`

**Permission buttons:**
- "Create Skill" button — show if `availableActions.canCreateSkill`
- "Create Namespace" button — show if `availableActions.canCreateNamespace`
- "Admin" link — show if `availableActions.canAccessAdmin`

**Empty state:** "No skills found. Be the first to publish one!"

---

## Skill detail page

**Endpoint:** `GET /api/v1/frontend/skills/{namespace}/{slug}`

**Data returned:**
- `skill` — display name, owner, summary, visibility, status, stats
- `versions` — list of versions with status
- `files` — package files
- `availableActions` — `canEdit`, `canPublish`, `canDelete`, `canSubmitForReview`, `canRequestPromotion`, `canStar`, `canReport`, `canManage`

**Recommended loading order:**
1. Fetch `GET /api/v1/frontend/skills/{namespace}/{slug}`
2. If user is logged in, also fetch `GET /api/v1/skills/{namespace}/{slug}/releases/latest`
3. Optionally fetch `GET /api/v1/skills/{namespace}/{slug}/issues` for recent issues

**Permission buttons:**

| Button | Condition |
|---|---|
| Edit | `availableActions.canEdit` |
| Delete | `availableActions.canDelete` |
| Star | `availableActions.canStar` (always visible, toggle state) |
| Report | `availableActions.canReport` |
| Submit for review | `availableActions.canSubmitForReview` |
| Request promotion | `availableActions.canRequestPromotion` |
| Publish new version | `availableActions.canPublish` |
| Manage settings | `availableActions.canManage` |

**Tabs/sub-pages:**
- **Versions tab** — use versions from the read model; link to version detail
- **Releases tab** — call `GET /api/v1/frontend/skills/{namespace}/{slug}/releases`
- **Community tab** — call `GET /api/v1/frontend/skills/{namespace}/{slug}/issues` for issues, `.../discussions` for discussions, `.../wiki` for wiki
- **CI tab** — call `GET /api/v1/skills/{skillId}/ci/runs` (need skillId from skill data)

**Empty states:**
- No versions: "This skill has no published versions yet."
- No releases: "No releases yet."
- No CI runs: "No CI pipeline runs yet."

---

## Namespace page

**Endpoint:** `GET /api/v1/frontend/namespaces/{slug}`

**Data returned:**
- `namespace` — slug, display name, type, description
- `members` — list of members with roles
- `availableActions` — `canEdit`, `canDelete`, `canManageMembers`, `canTransferOwner`, `canLeave`, `canJoin`

**Recommended loading order:**
1. Fetch `GET /api/v1/frontend/namespaces/{slug}`
2. Optionally list skills in namespace via search

**Permission buttons:**

| Button | Condition |
|---|---|
| Edit | `availableActions.canEdit` |
| Delete | `availableActions.canDelete` |
| Manage members | `availableActions.canManageMembers` |
| Transfer ownership | `availableActions.canTransferOwner` |
| Leave | `availableActions.canLeave` |
| Join | `availableActions.canJoin` |

---

## Release page

**List endpoint:** `GET /api/v1/frontend/skills/{namespace}/{slug}/releases`
**Detail endpoint:** `GET /api/v1/frontend/skills/{namespace}/{slug}/releases/{releaseID}`

**List data:**
- `releases` — list with id, versionId, channel, title, draft, prerelease, yanked, publishedAt, publisherId
- `totalCount`, `page`, `size`
- `availableActions` — `canCreateRelease`

**Detail data:**
- `release` — full release detail
- `assets` — downloadable assets
- `availableActions` — `canEdit`, `canDelete`, `canYank`, `canUnYank`

**Permission buttons (detail):**

| Button | Condition |
|---|---|
| Edit (title/notes) | `availableActions.canEdit` |
| Delete | `availableActions.canDelete` |
| Yank | `availableActions.canYank` |
| Unyank | `availableActions.canUnYank` |
| Publish draft | Available if `release.draft === true` and user has permission |
| Create release | `availableActions.canCreateRelease` (list page) |

**Mutations:**
- Create: `POST /api/v1/skills/{namespace}/{slug}/releases`
- Update: `PATCH /api/v1/skills/{namespace}/{slug}/releases/{releaseID}`
- Publish: `POST /api/v1/skills/{namespace}/{slug}/releases/{releaseID}/publish`
- Delete: `DELETE /api/v1/skills/{namespace}/{slug}/releases/{releaseID}`

**Empty state:** "No releases yet. Create a release from a published version."

---

## Community page

The community section groups issues, discussions, wiki, and change proposals for a skill.

### Issue list

**Endpoint:** `GET /api/v1/frontend/skills/{namespace}/{slug}/issues`

**Data:** `issues[]`, `totalCount`, `page`, `size`, `availableActions.canCreateIssue`

**Empty state:** "No issues reported. Found a bug? Create an issue."

### Issue detail

**Endpoint:** `GET /api/v1/frontend/skills/{namespace}/{slug}/issues/{issueID}`

**Data:** `issue`, `comments[]`, `availableActions.canEdit/canDelete/canClose/canReopen`

### Discussion list

**Endpoint:** `GET /api/v1/frontend/skills/{namespace}/{slug}/discussions`

**Data:** `discussions[]`, `totalCount`, `page`, `size`, `availableActions.canCreateDiscussion`

### Discussion detail

**Endpoint:** `GET /api/v1/frontend/skills/{namespace}/{slug}/discussions/{discussionID}`

**Data:** `discussion`, `comments[]`, `availableActions` — `canEdit`, `canDelete`, `canLock`, `canPin`, `canAcceptAnswer`

### Wiki list / detail

**List:** `GET /api/v1/frontend/skills/{namespace}/{slug}/wiki`
**Detail:** `GET /api/v1/frontend/skills/{namespace}/{slug}/wiki/{pageSlug}`

### Proposal list / detail

**List:** `GET /api/v1/frontend/skills/{namespace}/{slug}/proposals`
**Detail:** `GET /api/v1/frontend/skills/{namespace}/{slug}/proposals/{proposalID}`

---

## Review/Admin page

### Review queue

**Endpoint:** `GET /api/v1/frontend/reviews`

**Query:** `?page=0&size=20` (defaults shown; `size` capped at 100)

**Data:** `tasks[]`, `pendingCount`, `page`, `size`, `hasMore`, `availableActions.canReview/canSubmit/canWithdraw`

**Pagination:** Returns at most `size` tasks. When `hasMore` is true, increment `page` to load the next window.

### Review detail

**Endpoint:** `GET /api/v1/frontend/reviews/{id}`

**Data:** `task`, `skillName`, `version`, `availableActions.canApprove/canReject/canWithdraw`

**Read-model only:** `/api/v1/frontend/reviews` and `/api/v1/frontend/reviews/{id}` are read-model endpoints. They expose the queue, detail, and viewer action flags, but the project currently does **not** provide HTTP endpoints to approve, reject, or withdraw a review task. If you need to implement review mutations, add separate HTTP routes that reuse the SDK `review` service and `agentci` gate enforcement; do not assume `POST /api/v1/frontend/reviews/{id}/approve` or similar exists today.

### Promotion queue/detail

**Endpoints:**
- `GET /api/v1/frontend/promotions`
- `GET /api/v1/frontend/promotions/{id}`

**Query (queue):** `?page=0&size=20` (defaults shown; `size` capped at 100)

**Data:** `requests[]`, `pendingCount`, `page`, `size`, `hasMore`, `availableActions.canReview/canSubmit/canWithdraw` (queue); `request`, `sourceSkillName`, `availableActions.canApprove/canReject/canWithdraw` (detail)

**Pagination:** Returns at most `size` requests. When `hasMore` is true, increment `page` to load the next window.

**Read-model only:** promotion frontend endpoints are also read-model only. There are currently no HTTP endpoints to approve, reject, or withdraw a promotion request. Promotion mutations must be implemented as separate HTTP routes backed by the SDK `promotion` service.

### Admin dashboard

**Endpoint:** `GET /api/v1/frontend/admin`

**Data:** `stats` (totalSkills, totalNamespaces, totalUsers, pendingReviews, pendingPromotions, openReports), `availableActions` (7 boolean flags)

### Governance workbench

**Endpoint:** `GET /api/v1/frontend/governance`

**Data:** `summary`, `recentActivity[]`, `availableActions` — `canReview`, `canAccessAdmin`, `canViewAuditLog`

---

## Agent CI page

**Note:** There is no dedicated frontend read-model for Agent CI. Use the portal routes directly.

**List pipeline runs:** `GET /api/v1/skills/{skillID}/ci/runs`

**Pipeline run detail:** `GET /api/v1/skills/{skillID}/ci/runs/{runID}`

**Check runs:** `GET /api/v1/skills/{skillID}/ci/runs/{runID}/checks`

**Check detail:** `GET /api/v1/skills/{skillID}/ci/checks/{checkID}`

**Artifacts:** `GET /api/v1/skills/{skillID}/ci/checks/{checkID}/artifacts`

**Gate evaluation:** `GET /api/v1/skills/{skillID}/ci/gates?trigger=publish&versionId=5`

**Recommended loading order:**
1. Fetch pipeline runs list
2. Click a run → fetch run detail + check runs
3. Click a check → fetch check detail + artifacts

**Empty states:**
- No runs: "No CI pipeline runs yet. CI runs are triggered when a version is published."
- Run status colors: PENDING (gray), RUNNING (blue), COMPLETED (green/red based on pass/fail), FAILED (red)
- Gate failed: Red badge showing which policies failed

---

## Summary of frontend routes

| Page | Endpoint |
|---|---|
| Home / Discover | `GET /api/v1/frontend/search` |
| Skill detail | `GET /api/v1/frontend/skills/{ns}/{slug}` |
| Version detail | `GET /api/v1/frontend/skills/{ns}/{slug}/versions/{v}` |
| Publish | `GET /api/v1/frontend/skills/{ns}/publish/validate` |
| Namespace list | `GET /api/v1/frontend/namespaces` |
| Namespace detail | `GET /api/v1/frontend/namespaces/{slug}` |
| Release list | `GET /api/v1/frontend/skills/{ns}/{slug}/releases` |
| Release detail | `GET /api/v1/frontend/skills/{ns}/{slug}/releases/{id}` |
| Issue list | `GET /api/v1/frontend/skills/{ns}/{slug}/issues` |
| Issue detail | `GET /api/v1/frontend/skills/{ns}/{slug}/issues/{id}` |
| Discussion list | `GET /api/v1/frontend/skills/{ns}/{slug}/discussions` |
| Discussion detail | `GET /api/v1/frontend/skills/{ns}/{slug}/discussions/{id}` |
| Wiki list | `GET /api/v1/frontend/skills/{ns}/{slug}/wiki` |
| Wiki detail | `GET /api/v1/frontend/skills/{ns}/{slug}/wiki/{pageSlug}` |
| Proposal list | `GET /api/v1/frontend/skills/{ns}/{slug}/proposals` |
| Proposal detail | `GET /api/v1/frontend/skills/{ns}/{slug}/proposals/{id}` |
| Review queue | `GET /api/v1/frontend/reviews` |
| Review detail | `GET /api/v1/frontend/reviews/{id}` |
| Promotion queue | `GET /api/v1/frontend/promotions` |
| Promotion detail | `GET /api/v1/frontend/promotions/{id}` |
| Governance | `GET /api/v1/frontend/governance` |
| Admin | `GET /api/v1/frontend/admin` |
