# TypeScript SDK

The TypeScript client lives at `clients/typescript/skillhub/`. It provides typed wrappers over every SkillHub HTTP endpoint.

## Installation / local reference

Since the SDK is not published to npm, reference it locally:

```json
// package.json (in your frontend project)
{
  "dependencies": {
    "@miqro/skillhub-client": "file:../clients/typescript/skillhub"
  }
}
```

Or build it first and use the dist:

```bash
cd clients/typescript/skillhub
npm install
npm run build
```

## Initializing the client

```typescript
import { SkillHubClient } from "@miqro/skillhub-client";

// Default: http://localhost:8080
const client = new SkillHubClient();

// Custom base URL (string form — backward compatible)
const client = new SkillHubClient("https://skillhub.example.com");

// Full options object
const client = new SkillHubClient({
  baseUrl: "https://skillhub.example.com",
  credentials: "include",        // cookie-based session auth
  token: "sk_abc123",            // static bearer token
  getToken: () => localStorage.getItem("token") ?? undefined, // dynamic token
  headers: { "X-Client": "web" }, // custom headers merged into every request
  fetch: customFetch,            // custom fetch (SSR, tests, polyfills)
});
```

The old `new SkillHubClient()` and `new SkillHubClient("http://...")` forms continue to work.

## Auth

### Session cookie mode

```typescript
const client = new SkillHubClient({
  baseUrl: "http://localhost:8080",
  credentials: "include",
});

const { data: user } = await client.login("username", "password");
// Cookie is set; subsequent requests use the session.
```

### Bearer token mode

```typescript
// Static token
const client = new SkillHubClient({
  baseUrl: "https://skillhub.example.com",
  token: "sk_abc123",
});

// Dynamic token (e.g. from localStorage, refreshed on each call)
const client = new SkillHubClient({
  baseUrl: "https://skillhub.example.com",
  getToken: () => localStorage.getItem("skillhub_token") ?? undefined,
});
```

Every request automatically sends `Authorization: Bearer <token>`. Static `token` takes precedence over `getToken`. If `getToken` returns `undefined`, no auth header is sent for that request.

### Custom fetch for SSR/tests

```typescript
const client = new SkillHubClient({
  baseUrl: "http://localhost:8080",
  fetch: nodeFetch, // e.g. 'node-fetch' in server-side code
});
```

## Error handling

### Envelope mode (compatible — existing code unchanged)

Every method returns `Envelope<T>`:

```typescript
const result = await client.search({ keyword: "test" });
if (result.success && result.data) {
  // use result.data
} else {
  console.error(result.error?.code, result.error?.message);
}
```

### Unwrap mode (throwing)

```typescript
import { SkillHubClient, SkillHubError } from "@miqro/skillhub-client";

const client = new SkillHubClient({
  baseUrl: "http://localhost:8080",
  credentials: "include",
});

try {
  const me = await client.unwrap(client.me());
  console.log(me.displayName);
} catch (err) {
  if (err instanceof SkillHubError) {
    // err.code    — "not_found", "unauthorized", "client.network_error", etc.
    // err.message — human-readable message
    // err.status  — HTTP status code (e.g. 404)
    // err.details — optional server-provided details
    console.error(err.code, err.status, err.message);
  }
}
```

Error codes:
| Code | Meaning |
|---|---|
| `client.network_error` | fetch() rejected (network down, DNS, CORS) |
| `client.invalid_json` | Response body was not valid JSON |
| `client.error` | Error envelope without a specific code |
| `<server code>` | Server-provided code (e.g. `not_found`, `unauthorized`, `forbidden`) |

## Pagination iterators

Bounded async iterators prevent unbounded data fetching. Every iterator defaults to `maxPages: 10`.

```typescript
// Review queue
for await (const page of client.iterFrontendReviews({ size: 50, maxPages: 5 })) {
  for (const task of page.tasks) {
    console.log(task.skillName, task.status);
  }
}

// Search with filters
for await (const page of client.iterFrontendSearch({
  keyword: "agent",
  sortBy: "downloads",
  size: 20,
  maxPages: 3,
})) {
  console.log(page.searchResult.skillIds);
}

// Community issues
for await (const page of client.iterFrontendIssues("ns", "skill", { size: 50 })) {
  for (const issue of page.issues) {
    console.log(issue.title);
  }
}

// Releases
for await (const page of client.iterReleases("ns", "skill", { maxPages: 5 })) {
  console.log(page.releases);
}

// Pipeline runs
for await (const page of client.iterPipelineRuns(1, { size: 20 })) {
  console.log(page.runs);
}
```

Each iterator stops when:
- `maxPages` is reached;
- `hasMore` is `false` (review/promotion queues);
- the returned item list is empty;
- `totalCount` indicates no more results.

Iterator options:
```typescript
interface PageIteratorOptions {
  page?: number;    // starting page (default 0)
  size?: number;    // page size (default 20)
  maxPages?: number; // max pages to fetch (default 10)
}
```

Available iterators:
- `iterFrontendSearch(query)`
- `iterFrontendReviews(options?)`
- `iterFrontendPromotions(options?)`
- `iterFrontendIssues(namespace, slug, options?)`
- `iterFrontendDiscussions(namespace, slug, options?)`
- `iterFrontendProposals(namespace, slug, options?)`
- `iterReleases(namespace, slug, options?)`
- `iterPipelineRuns(skillId, options?)`

## Search skills

```typescript
const { data } = await client.search({
  keyword: "agent",
  sortBy: "downloads",
  installableOnly: true,
});

console.log(data.skillIds);  // [1, 5, 12]
console.log(data.total);     // 3
```

## Get skill detail

```typescript
const { data } = await client.getSkill("my-namespace", "my-skill");

console.log(data.displayName);  // "My Skill"
console.log(data.canManage);    // true | false
```

## Get releases

```typescript
// List releases
const { data: releases } = await client.listReleases("my-namespace", "my-skill");

// Get latest stable
const { data: latest } = await client.getLatestRelease("my-namespace", "my-skill");
console.log(latest.channel);  // "stable"

// Get a specific release
const { data: detail } = await client.getRelease("my-namespace", "my-skill", 1);
console.log(detail.release.title);
console.log(detail.assets.length);

// Create a release (always draft)
const { data: created } = await client.createRelease("my-namespace", "my-skill", {
  versionId: 5,
  title: "v1.0.0",
  notes: "First stable release",
});

// Publish a draft release (runs CI gate enforcement)
const { data: published } = await client.publishRelease("my-namespace", "my-skill", 1);

// Update release metadata
await client.updateRelease("my-namespace", "my-skill", 1, {
  title: "Updated release title",
});

// Delete release
await client.deleteRelease("my-namespace", "my-skill", 1);
```

## Get agent CI runs and checks

```typescript
const skillId = 1;

// List pipeline runs
const { data: runs } = await client.listPipelineRuns(skillId);
console.log(runs.runs[0].status);  // "COMPLETED"

// Get a specific run
const { data: run } = await client.getPipelineRun(skillId, runId);

// List check runs for a pipeline run
const { data: checks } = await client.listCheckRuns(skillId, runId);

// Get a specific check
const { data: check } = await client.getCheckRun(skillId, checkId);

// List check artifacts
const { data: artifacts } = await client.listCheckArtifacts(skillId, checkId);

// Evaluate CI gates
const { data: gates } = await client.evaluateGates(skillId, {
  trigger: "publish",
  versionId: 5,
});
console.log(gates.passed);  // true | false
```

## Frontend page read models

Each frontend route returns a read model with `availableActions`:

```typescript
// Home/search page
const { data: home } = await client.frontendSearch({
  keyword: "agent",
  sortBy: "downloads",
  page: 0,
  size: 20,
  labelSlugs: ["go"],
  installableOnly: true,
});
// home.searchResult.skillIds
// home.availableActions.canCreateSkill

// Skill detail page
const { data: detail } = await client.frontendSkillDetail("ns", "my-skill");
// detail.skill, detail.versions, detail.files
// detail.availableActions.canEdit, canPublish, canStar, ...

// Release list page
const { data: releasePage } = await client.frontendReleaseList("ns", "my-skill");
// releasePage.availableActions.canCreateRelease

// Release detail page
const { data: releaseDetail } = await client.frontendReleaseDetail("ns", "my-skill", 1);

// Namespace detail
const { data: ns } = await client.frontendNamespaceDetail("my-ns");
// ns.namespace, ns.members, ns.availableActions

// Review queue
const { data: reviews } = await client.frontendReviews();
// reviews.tasks (with skill/version/namespace enrichment), reviews.pendingCount
// reviews.availableActions.canReview

// Review detail
const { data: reviewDetail } = await client.frontendReviewDetail(1);
// reviewDetail.task, reviewDetail.skillName, reviewDetail.version
// reviewDetail.availableActions.canApprove, canReject, canWithdraw

// Promotion queue
const { data: promotions } = await client.frontendPromotions();
// promotions.requests (with source/target enrichment), promotions.pendingCount
// promotions.availableActions.canReview

// Promotion detail
const { data: promotionDetail } = await client.frontendPromotionDetail(1);
// promotionDetail.request, promotionDetail.sourceSkillName
// promotionDetail.availableActions.canApprove, canReject, canWithdraw

// Governance workbench
const { data: governance } = await client.frontendGovernance();
// governance.summary.total, governance.summary.unread
// governance.summary.pendingReviews, governance.summary.pendingPromotions
// governance.recentActivity, governance.availableActions.canReview

// Admin page
const { data: admin } = await client.frontendAdmin();
// admin.stats.totalSkills, admin.stats.totalUsers, admin.stats.pendingReviews
// admin.availableActions.canManageSkills

// Community pages
const { data: issues } = await client.frontendIssueList("ns", "my-skill");
const { data: discussions } = await client.frontendDiscussionList("ns", "my-skill");
const { data: wiki } = await client.frontendWikiList("ns", "my-skill");
const { data: proposals } = await client.frontendProposalList("ns", "my-skill");
```

## Frontend read-model methods

All `/api/v1/frontend/*` methods return a typed `Envelope<T>` with `availableActions` computed for the authenticated viewer. Path parameters are URL-encoded automatically, so namespaces, slugs, version strings, and wiki page slugs can contain spaces or slashes.

```typescript
// Home / discover
const { data: home } = await client.frontendSearch({
  keyword: "agent",
  sortBy: "downloads",
  page: 0,
  size: 20,
  labelSlugs: ["go"],
  installableOnly: true,
});

// Skill detail page
const { data: skillPage } = await client.frontendSkillDetail("team-alpha", "example-skill");
if (skillPage.availableActions.canEdit) {
  // show edit button
}

// Version detail page
const { data: versionPage } = await client.frontendVersionDetail(
  "team-alpha", "example-skill", "1.0.0"
);

// Namespace detail page
const { data: nsPage } = await client.frontendNamespaceDetail("team-alpha");

// Release list/detail pages
const { data: releaseList } = await client.frontendReleaseList("team-alpha", "example-skill");
const { data: releaseDetail } = await client.frontendReleaseDetail(
  "team-alpha", "example-skill", 1
);

// Review and promotion queues (read-model only)
const { data: reviews } = await client.frontendReviews();
const { data: review } = await client.frontendReviewDetail(1);
const { data: promotions } = await client.frontendPromotions();
const { data: promotion } = await client.frontendPromotionDetail(1);

// Governance workbench and admin dashboard
const { data: governance } = await client.frontendGovernance();
const { data: admin } = await client.frontendAdmin();

// Community pages
const { data: issues } = await client.frontendIssueList("team-alpha", "example-skill");
const { data: issue } = await client.frontendIssueDetail("team-alpha", "example-skill", 1);
const { data: discussions } = await client.frontendDiscussionList("team-alpha", "example-skill");
const { data: discussion } = await client.frontendDiscussionDetail(
  "team-alpha", "example-skill", 1
);
const { data: wikiPages } = await client.frontendWikiList("team-alpha", "example-skill");
const { data: wikiPage } = await client.frontendWikiDetail(
  "team-alpha", "example-skill", "getting-started"
);
const { data: proposals } = await client.frontendProposalList("team-alpha", "example-skill");
const { data: proposal } = await client.frontendProposalDetail("team-alpha", "example-skill", 1);
```

## Review and promotion mutations

Review and promotion approve, reject, and withdraw are available as portal mutation methods:

```typescript
// Review mutations
await client.unwrap(client.approveReview(1, { comment: "Looks good" }));
await client.unwrap(client.rejectReview(2, { comment: "Needs more tests" }));
await client.unwrap(client.withdrawReview(3));

// Promotion mutations
await client.unwrap(client.approvePromotion(1, { comment: "Promoting" }));
await client.unwrap(client.rejectPromotion(2, { comment: "Not ready" }));
await client.unwrap(client.withdrawPromotion(3));
```

Backend authorization is enforced by the SDK — the client only passes credentials. SDK permission checks, gate enforcement, and status validation run server-side. After any mutation, refetch the frontend read model to get updated `availableActions`.

## Tool API (miqro CLI)

```typescript
// Workspace metadata
const { data: ws } = await client.toolWorkspaceMetadata();

// Package hash
const { data: hash } = await client.toolPackageHash([
  { path: "SKILL.md", content: "...", size: 100, contentType: "text/markdown" },
]);

// Resolve version
const { data: resolved } = await client.toolResolve("ns", "my-skill", "1.0.0");

// Install metadata
const { data: install } = await client.toolInstall("ns", "my-skill");

// Diff versions
const { data: diff } = await client.toolDiff("ns", "my-skill", "1.0.0", "2.0.0");

// Validate (with zip file)
const zipBlob = new Blob([...]);
const { data: validation } = await client.toolValidate("ns", zipBlob);

// Publish (with zip file)
const { data: published } = await client.toolPublish("ns", zipBlob);
```

## Community CRUD

```typescript
// Issues
const { data: issues } = await client.listIssues("ns", "my-skill", { status: "OPEN" });
await client.createIssue("ns", "my-skill", { title: "Bug report" });
await client.updateIssue("ns", "my-skill", 1, { status: "CLOSED" });
await client.addIssueComment("ns", "my-skill", 1, { body: "Fixed in v2." });

// Discussions
await client.createDiscussion("ns", "my-skill", { title: "Feature idea", category: "IDEAS" });

// Wiki
await client.createWikiPage("ns", "my-skill", {
  title: "Getting Started", slug: "getting-started", body: "# Hello"
});

// Proposals
await client.createProposal("ns", "my-skill", { title: "Refactor module X" });
```

## Maintenance status

The SDK is currently published as a single package entry point (`src/index.ts`) with generated and hand-maintained endpoint methods, typed envelopes, `unwrap()`, and pagination iterators. The `dist/` directory is intentionally not committed.

Future work (planned for Phase 25) will split the implementation into smaller domain modules (`auth`, `frontend`, `community`, `release`, `agentci`, `tooling`) while preserving the public `SkillHubClient` API and existing tests.

## API compatibility

- `new SkillHubClient()` — connects to `http://localhost:8080`
- `new SkillHubClient("http://...")` — custom base URL (backward compatible)
- `new SkillHubClient({ ... })` — full options (baseUrl, credentials, token, getToken, headers, fetch)
- Every method returns `Envelope<T>` (success/error envelope)
- Use `client.unwrap()` to get `T` directly with throwing `SkillHubError` on failure
- Bounded pagination iterators default to `maxPages: 10`

## Building and testing

```bash
cd clients/typescript/skillhub
npm install
npm run build         # tsc
npm test              # node --test dist/**/*.test.js
npx tsc --noEmit      # typecheck without emitting
```
