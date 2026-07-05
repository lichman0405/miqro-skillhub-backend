# Frontend Information Architecture

The information architecture (IA) and app shell contract for the future SkillHub web frontend.

This document is the blueprint that every frontend implementation — whether Vue, React, Svelte, or another stack — must follow. It defines pages, routes, data sources, and UX rules before a single component is written.

## Product Stance

The SkillHub frontend is a **skill-native** product, not a GitHub clone.

The primary object is a skill package version, not a git repository.

Every page, route, and component should serve one of these user goals:

- **Discover** — find useful skills by search, labels, sort, and installable filter.
- **Inspect** — understand a skill's manifest, versions, files, checks, and provenance.
- **Install** — download or resolve a concrete skill version.
- **Release** — create, review, publish, and withdraw skill releases.
- **Evaluate** — view CI pipeline runs, check results, artifacts, and gate outcomes.
- **Collaborate** — discuss, report issues, write wiki pages, and propose changes scoped to a skill.
- **Govern** — review submissions, approve/reject promotions, monitor activity, and manage admin tasks.

Git-style concepts (branches, commits, merge requests) may appear later as optional source integrations. The initial frontend must not require users to understand those concepts.

## App Shell

The app shell is the persistent layout that wraps every page. It defines the global structure without implementation code.

### Global Header

Always visible. Contains:

- **Logo / home link** — navigates to `/`.
- **Search bar** — full-text keyword search, submits to `/search`.
- **Namespace switcher** — dropdown of accessible namespaces. Selecting one scopes the current context. Default: all namespaces.
- **Create menu** — dropdown: New Skill, New Namespace, New Release (if context permits). Visibility driven by `availableActions`.
- **Auth state** — shows username and avatar when logged in, or "Sign in" link. After login, shows notification count badge that links to `/governance`.
- **Notifications indicator** — badge with unread count from `frontendGovernance().summary.unread`.

### Navigation

**Top-level (always visible):**

| Item | Route | Notes |
|---|---|---|
| Discover | `/` or `/search` | Home page |
| Namespaces | `/namespaces` | Browse/create namespaces |
| Reviews | `/reviews` | Badge: pending count. Hidden if not reviewer. |
| Promotions | `/promotions` | Badge: pending count. Hidden if not platform reviewer. |
| Governance | `/governance` | Role-aware workbench. Badge: unread. |
| Admin | `/admin` | SUPER_ADMIN only. Hidden otherwise. |

**Skill-level tabs (visible when on a skill page):**

These appear as a horizontal tab bar below the skill header on every route under `/skills/:namespace/:slug`:

| Tab | Route | Notes |
|---|---|---|
| Overview | `/skills/:namespace/:slug` | Skill detail, versions, files |
| Versions | `/skills/:namespace/:slug/versions/:version` | Version-centric view |
| Releases | `/skills/:namespace/:slug/releases` | Release list/detail |
| CI | `/skills/:namespace/:slug/ci` | Pipeline runs, checks |
| Issues | `/skills/:namespace/:slug/issues` | Issue list/detail |
| Discussions | `/skills/:namespace/:slug/discussions` | Discussion list/detail |
| Wiki | `/skills/:namespace/:slug/wiki` | Wiki pages |
| Proposals | `/skills/:namespace/:slug/proposals` | Change proposals |
| Settings | (future) | Skill settings |

### Action Areas

Every page has a role-aware **action area** — a toolbar or button group whose contents are driven exclusively by `availableActions` from the page's read model.

- Never hard-code role checks on the frontend.
- If `availableActions.canEdit` is true, show the Edit button. Otherwise hide it.
- Authorization is enforced server-side. The frontend renders UI hints; the backend denies unauthorized mutations.

### Error Boundary

Every page must be wrapped in an error boundary that catches:

- `SkillHubError` with `status` attached — render the appropriate error state (see Error and Empty States below).
- Network errors (`SkillHubError` with code `client.network_error`) — render a retryable "cannot reach server" state.
- Unexpected exceptions — render a generic fallback with a reload button.

### Loading Skeleton

Every page that fetches a read model should show a skeleton (pulsing placeholders matching the page layout) while the initial fetch is in flight. Subsequent navigations can use the cached read model.

### CORS / Same-Origin

Authenticated browser sessions require explicit `SKILLHUB_CORS_ALLOWED_ORIGINS` configuration on the backend. The frontend must be served from an allowed origin or the same origin as the backend.

## Route Inventory

Every route the frontend must support. Each is classified by priority, access, and capability.

### Top-Level Routes

| Route | Page | Priority | Access | Type |
|---|---|---|---|---|
| `/` | Discover / Home | MVP | Public | Read-model only |
| `/search` | Search results | MVP | Public | Read-model only |
| `/namespaces` | Namespace list | MVP | Public | Read-model only |
| `/namespaces/:namespace` | Namespace detail | MVP | Public | Read-model only |
| `/login` | Login page | MVP | Public (unauthenticated only) | Auth |
| `/settings/profile` | User profile settings | Post-MVP | Auth-only | Mutation-capable |
| `/settings/tokens` | API token management | Post-MVP | Auth-only | Mutation-capable |

### Skill Routes

| Route | Page | Priority | Access | Type |
|---|---|---|---|---|
| `/skills/:namespace/:slug` | Skill detail (overview) | MVP | Public | Read-model only |
| `/skills/:namespace/:slug/versions/:version` | Version detail | MVP | Public | Read-model only |
| `/skills/:namespace/:slug/releases` | Release list | MVP | Public | Read-model only |
| `/skills/:namespace/:slug/releases/:releaseId` | Release detail | MVP | Public | Read-model only |
| `/skills/:namespace/:slug/ci` | CI pipeline runs | MVP | Public | Read-model only |
| `/skills/:namespace/:slug/issues` | Issue list | MVP | Public | Read-model only |
| `/skills/:namespace/:slug/issues/:issueId` | Issue detail | MVP | Public | Mutation-capable |
| `/skills/:namespace/:slug/discussions` | Discussion list | MVP | Public | Read-model only |
| `/skills/:namespace/:slug/discussions/:discussionId` | Discussion detail | MVP | Public | Mutation-capable |
| `/skills/:namespace/:slug/wiki` | Wiki page list | MVP | Public | Read-model only |
| `/skills/:namespace/:slug/wiki/:pageSlug` | Wiki page detail | MVP | Public | Mutation-capable |
| `/skills/:namespace/:slug/proposals` | Proposal list | MVP | Public | Read-model only |
| `/skills/:namespace/:slug/proposals/:proposalId` | Proposal detail | MVP | Public | Mutation-capable |

### Work Queue Routes

| Route | Page | Priority | Access | Type |
|---|---|---|---|---|
| `/reviews` | Review queue | MVP | Auth-only (reviewer) | Read-model only |
| `/reviews/:reviewId` | Review detail | MVP | Auth-only (reviewer) | Mutation-capable |
| `/promotions` | Promotion queue | MVP | Auth-only (platform reviewer) | Read-model only |
| `/promotions/:promotionId` | Promotion detail | MVP | Auth-only (platform reviewer) | Mutation-capable |
| `/governance` | Governance workbench | MVP | Auth-only | Read-model only |
| `/admin` | Admin dashboard | Post-MVP | Admin-only (SUPER_ADMIN) | Read-model only |

### Route Priority Legend

- **MVP** — Must be implemented in the first frontend build phase.
- **Post-MVP** — Can be deferred to a later frontend build phase.
- **Admin-only** — Only visible to SUPER_ADMIN. Can be a simple page initially.
- **Auth-only** — Requires authentication. Redirect to `/login` if unauthenticated.
- **Read-model only** — The page only fetches data. No mutations happen on this route.
- **Mutation-capable** — The page contains buttons/forms that call mutation endpoints.

## Page-To-SDK Matrix

Every page maps to specific `SkillHubClient` methods. This matrix is the contract between frontend pages and the TypeScript SDK.

| Page | Read method | Mutation / extra methods | Notes |
|---|---|---|---|
| Discover | `frontendSearch(query)` | none | Pass `keyword`, `sortBy`, `labelSlugs`, `installableOnly`, `page`, `size`. Use `availableActions` for "Create Skill" / "Create Namespace" buttons. |
| Namespace list | `frontendNamespaces()` | none | Lists ACTIVE namespaces. |
| Namespace detail | `frontendNamespaceDetail(slug)` | none | Shows namespace info, members, and `availableActions` for management buttons. |
| Skill detail | `frontendSkillDetail(ns, slug)` | none | Shows skill info, versions, files, `availableActions`. Link to version/release/community tabs. |
| Version detail | `frontendVersionDetail(ns, slug, version)` | none | Version-centered quality view. Show version status, publish date, file list. |
| Release list | `frontendReleaseList(ns, slug)` | none | List of releases with channel, status, dates. |
| Release detail | `frontendReleaseDetail(ns, slug, id)` | `createRelease`, `updateRelease`, `deleteRelease`, `publishRelease` | After any mutation, refetch `frontendReleaseDetail`. Use `availableActions` for edit/delete/yank/publish buttons. |
| CI pipeline runs | `listPipelineRuns(skillId, page, size)` | `getPipelineRun`, `listCheckRuns`, `getCheckRun`, `listCheckArtifacts`, `evaluateGates` | Portal API — no frontend read model. Need `skillId` from skill detail. |
| Issue list | `frontendIssueList(ns, slug, page, size)` | none | Skill-scoped issue list. |
| Issue detail | `frontendIssueDetail(ns, slug, id)` | `createIssue`, `updateIssue`, `deleteIssue`, `addIssueComment` | After mutation, refetch `frontendIssueDetail`. |
| Discussion list | `frontendDiscussionList(ns, slug, page, size)` | none | Skill-scoped discussion list. |
| Discussion detail | `frontendDiscussionDetail(ns, slug, id)` | `createDiscussion`, `updateDiscussion`, `deleteDiscussion`, `addDiscussionComment`, `acceptAnswer` | After mutation, refetch `frontendDiscussionDetail`. |
| Wiki list | `frontendWikiList(ns, slug)` | none | Skill-scoped wiki index. |
| Wiki page detail | `frontendWikiDetail(ns, slug, pageSlug)` | `createWikiPage`, `updateWikiPage`, `listWikiVersions` | After mutation, refetch `frontendWikiDetail`. |
| Proposal list | `frontendProposalList(ns, slug, page, size)` | none | Skill-scoped proposal list. |
| Proposal detail | `frontendProposalDetail(ns, slug, id)` | `createProposal`, `updateProposal` | After mutation, refetch `frontendProposalDetail`. |
| Review queue | `frontendReviews(page, size)` | none | Paginated queue with `hasMore`. |
| Review detail | `frontendReviewDetail(id)` | `approveReview`, `rejectReview`, `withdrawReview` | Portal mutations. After mutation, refetch `frontendReviewDetail` (and queue if returning to it). |
| Promotion queue | `frontendPromotions(page, size)` | none | Paginated queue with `hasMore`. |
| Promotion detail | `frontendPromotionDetail(id)` | `approvePromotion`, `rejectPromotion`, `withdrawPromotion` | Portal mutations. After mutation, refetch `frontendPromotionDetail` (and queue if returning to it). |
| Governance | `frontendGovernance()` | none | Role-aware workbench: summary counts, recent activity, `availableActions`. |
| Admin | `frontendAdmin()` | none | SUPER_ADMIN only. Aggregate stats. Unauthorized viewers receive zero stats. |
| Login | `login(username, password)` | `me`, `logout` | Session cookie or bearer token. After login, refetch `me` to populate auth state. |
| Settings/Profile | `me()` | user profile mutations (future) | Read current user info. |
| Settings/Tokens | none | API token CRUD (future) | Manage personal access tokens. |

## Data Loading Rules

These rules apply to every page and every data fetch:

1. **Prefer `SkillHubClient` over raw `fetch`.** The SDK handles auth, URL encoding, error normalization, and pagination consistently.

2. **Use `client.unwrap(...)` for page loaders.** Page loaders that must return data or redirect should use `unwrap()` so thrown `SkillHubError` propagates to the error boundary with a typed `status` and `code`.

3. **Use envelope mode for inline error display.** When the UI needs to show server error details inline (e.g., a validation message next to a form field), use envelope mode (`const result = await client.someMethod(...)`) and inspect `result.error`.

4. **Use SDK iterators only for bounded background prefetch.** Iterators (`iterFrontendReviews`, `iterFrontendSearch`, etc.) are designed for bounded sequential access. Use them for caching or pre-warming, not for rendering the primary page data.

5. **For user-driven paging, call page methods directly.** Pass explicit `page` and `size` parameters. Do not use iterators for "Next Page" buttons — call the underlying method with incremented `page`.

6. **Never prefetch unbounded queues.** Always set `maxPages` on iterators. Review and promotion queues cap `size` at 100 server-side; use that bound.

7. **Do not derive authorization from frontend role strings.** Render actions from `availableActions` booleans. The backend enforces mutations. If the frontend shows a button for an action the user cannot actually perform, the backend returns 403.

8. **After any mutation, refetch the relevant frontend read model.** A mutation (approve, reject, create, update, delete) changes server state. The read model that was fetched before the mutation is stale. Always refetch the page's read model after a successful mutation to get updated data and `availableActions`.

9. **Import only from `@miqro/skillhub-client`.** Do not import from internal paths like `@miqro/skillhub-client/dist/domains/auth` or `@miqro/skillhub-client/dist/types/common`. These paths are not part of the public compatibility contract and may change without notice.

## State Model

Recommended frontend state buckets. Any framework store (Vue Query / TanStack Query, Pinia, Zustand, Redux, or framework-native loaders) can implement these:

| Bucket | Contents | Lifetime |
|---|---|---|
| Session / auth | Current `Principal` from `me()`, auth method (cookie/bearer), login state | Until logout or session expiry |
| Current route params | `namespace`, `slug`, `version`, `issueId`, etc. from URL | Duration of page visit |
| Page read model cache | Last fetched read model per page (with `availableActions`) | Invalidated on mutation or navigation |
| Transient mutation state | In-flight mutation status, optimistic updates, pending comments/edits | Duration of mutation + refetch |
| Notifications / toasts | Success/error toasts, notification count from governance summary | Displayed and dismissed |
| User preferences | Theme, default page size, sidebar collapsed state | Persisted to `localStorage` |

Read-model cache keys should include the viewer identity — a different user sees different `availableActions`. Invalidate the cache on login, logout, and after any mutation on that resource.

## Error And Empty States

### HTTP Error States

| Status | Cause | Frontend behavior |
|---|---|---|
| 401 | Missing or expired credentials | Redirect to `/login`. Preserve the intended destination for post-login redirect. |
| 403 | Authenticated but not authorized | Show "You don't have permission to do that." with context about which action was denied. Do not redirect to login. |
| 404 | Route, skill, version, or resource not found | Show "Not found" page with a link back to Discover. Distinguish between "skill not found" (bad slug) and "version not found" (bad version string). |
| 409 | Conflict — e.g., gate enforcement failed, duplicate resource | Show the server-provided conflict reason. For gate failures, show which policies did not pass. |
| 422 / 400 | Validation error — bad input | Show validation feedback inline next to the relevant form field when possible. Fall back to a summary banner. |
| 503 | Backend unavailable or starting | Show "Server starting..." or "Service unavailable" with a retry button. Do not auto-redirect. |
| Network error | fetch() rejected (DNS, CORS, connection refused) | Show "Cannot reach server. Check your connection." with a retry button. |

### Empty States

| Scenario | Message | CTA |
|---|---|---|
| No search results | "No skills found." | "Create your first skill" (if `canCreateSkill`) |
| Skill has no versions | "This skill has no published versions yet." | "Publish a version" (if `canPublish`) |
| No releases | "No releases yet." | "Create a release from a published version" (if `canCreateRelease`) |
| No CI pipeline runs | "No CI pipeline runs yet." | "Publish a version to trigger CI." |
| No issues | "No issues reported." | "Create an issue" (if `canCreateIssue`) |
| No discussions | "No discussions yet." | "Start a discussion" (if `canCreateDiscussion`) |
| No wiki pages | "No wiki pages yet." | "Create the first page" (if `canEdit`) |
| No proposals | "No change proposals yet." | "Create a proposal" (if permission allows) |
| Review queue empty | "All caught up! No pending reviews." | No CTA needed. |
| Promotion queue empty | "All caught up! No pending promotions." | No CTA needed. |
| Governance — no activity | "No recent activity." | No CTA needed. |
| Admin — no data | Stats show zero counts. | No CTA needed (unauthenticated viewers get zero stats). |

## Future Frontend Build Phases

This sequence is guidance for whoever builds the UI, not the backend phase list.

1. **App shell, routing, auth, SDK client provider.**
   - Set up the framework project (recommended: Vue 3 + Vite + TypeScript).
   - Implement the global header, navigation, and error boundary.
   - Create the `SkillHubClient` provider (supply `baseUrl`, `credentials`, `getToken`).
   - Implement login page and auth state management.
   - Implement route guards (redirect to `/login` on 401, show 403 on forbidden).
   - Wire the navigation to show/hide links based on auth state.

2. **Discover, namespace, skill detail, version detail.**
   - Home/discover page with search, labels, sort, and installable filters.
   - Namespace list and detail pages.
   - Skill detail page with versions tab, files list, and `availableActions`-driven action buttons.
   - Version detail page.
   - Loading skeletons and empty states for each.

3. **Releases, CI, install/download flows.**
   - Release list and detail pages.
   - Release create/edit/publish/yank workflows.
   - CI pipeline runs page with run/check/artifact drill-down.
   - Gate evaluation display.
   - Download/install buttons that use the tool API resolve/install methods.

4. **Community: issues, discussions, wiki, proposals.**
   - Issue list/detail with create/edit/comment workflows.
   - Discussion list/detail with create/edit/comment/accept-answer workflows.
   - Wiki page list/detail with create/edit and version history.
   - Proposal list/detail with create/update workflows.
   - Community search across issues, discussions, wiki.

5. **Review, promotion, governance, admin workbenches.**
   - Review queue and detail with approve/reject/withdraw workflows.
   - Promotion queue and detail with approve/reject/withdraw workflows.
   - Governance workbench with activity feed and summary counts.
   - Admin dashboard (SUPER_ADMIN only).

6. **Polish: accessibility, keyboard navigation, responsive layout, observability.**
   - WCAG 2.1 AA compliance.
   - Keyboard navigation for all interactive elements.
   - Responsive layout (mobile-first where practical).
   - Client-side error tracking and performance monitoring.

### Recommended Default Stack

The recommended default stack for the eventual frontend is **Vue 3 + Vite + TypeScript**, consistent with the original target architecture and the existing TypeScript SDK. The committed frontend contract source is the TypeScript SDK plus the guides, not raw HTTP guesses.

Future app work should start from this guide and `guides/frontend-integration.md`.
