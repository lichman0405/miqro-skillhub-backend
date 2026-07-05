# @miqro/skillhub-client

TypeScript client for the SkillHub HTTP API.

## Usage

```typescript
import { SkillHubClient } from "@miqro/skillhub-client";

// Default base URL (http://localhost:8080)
const client = new SkillHubClient();

// Custom base URL
const client = new SkillHubClient("https://skillhub.example.com");

// Full options
const client = new SkillHubClient({
  baseUrl: "https://skillhub.example.com",
  credentials: "include",        // cookie-based auth
  token: "sk_...",               // static bearer token
  getToken: () => localStorage.getItem("token") ?? undefined, // dynamic token
  headers: { "X-Client": "web" }, // custom headers
  fetch: customFetch,            // custom fetch (SSR, tests, polyfills)
});
```

## Auth

### Session cookie mode

```typescript
const client = new SkillHubClient({
  baseUrl: "http://localhost:8080",
  credentials: "include",
});

const { data: user } = await client.login("alice", "password123");
// Cookie is set; subsequent requests use the session.
```

### Bearer token mode

```typescript
const client = new SkillHubClient({
  baseUrl: "https://skillhub.example.com",
  token: "sk_abc123",
});
// Every request sends Authorization: Bearer sk_abc123
```

### Dynamic token (e.g. localStorage)

```typescript
const client = new SkillHubClient({
  baseUrl: "https://skillhub.example.com",
  getToken: () => localStorage.getItem("skillhub_token") ?? undefined,
});
```

## Error handling

### Envelope mode (compatible)

Every method returns `Envelope<T>`:

```typescript
const result = await client.search({ keyword: "agent" });
if (result.success && result.data) {
  console.log(result.data.skillIds);
} else {
  console.error(result.error?.code, result.error?.message);
}
```

### Unwrap mode (throwing)

```typescript
try {
  const data = await client.unwrap(client.search({ keyword: "agent" }));
  console.log(data.skillIds);
} catch (err) {
  if (err instanceof SkillHubError) {
    console.error(err.code, err.message, err.status);
  }
}
```

`SkillHubError` includes `code`, `message`, `status` (HTTP status), `details`, and `response`.

## Pagination iterators

Bounded async iterators for list endpoints. Default: max 10 pages.

```typescript
// Review queue — stops at hasMore=false, empty list, or maxPages
for await (const page of client.iterFrontendReviews({ size: 50, maxPages: 5 })) {
  for (const task of page.tasks) {
    console.log(task.skillName);
  }
}

// Search
for await (const page of client.iterFrontendSearch({ keyword: "agent", maxPages: 3 })) {
  console.log(page.searchResult.skillIds);
}
```

Available iterators:
- `iterFrontendSearch(query)`
- `iterFrontendReviews(options?)`
- `iterFrontendPromotions(options?)`
- `iterFrontendIssues(ns, slug, options?)`
- `iterFrontendDiscussions(ns, slug, options?)`
- `iterFrontendProposals(ns, slug, options?)`
- `iterReleases(ns, slug, options?)`
- `iterPipelineRuns(skillId, options?)`

## Frontend read-model examples

Every `/api/v1/frontend/*` route has a typed client method:

```typescript
const { data: home } = await client.frontendSearch({
  keyword: "agent", sortBy: "downloads", page: 0, size: 20,
});

const { data: skillPage } = await client.frontendSkillDetail("my-namespace", "my-skill");
if (skillPage.availableActions.canEdit) {
  // show edit button
}
```

### Community pages

```typescript
const { data: issues } = await client.frontendIssueList("ns", "skill", 0, 20);
const { data: issue } = await client.frontendIssueDetail("ns", "skill", 1);
const { data: wiki } = await client.frontendWikiList("ns", "skill");
const { data: page } = await client.frontendWikiDetail("ns", "skill", "getting-started");
```

Path parameters (namespace, slug, version, pageSlug) are URL-encoded automatically.

## Portal API examples

```typescript
// Search skills
const { data: results } = await client.search({ keyword: "agent", installableOnly: true });

// Get skill detail
const { data: skill } = await client.getSkill("my-namespace", "my-skill");

// Releases
const { data: releases } = await client.listReleases("ns", "skill", 0, 20);
const { data: latest } = await client.getLatestRelease("ns", "skill", "stable");

// Community
const { data: issues } = await client.listIssues("ns", "skill", { status: "OPEN" });
await client.createIssue("ns", "skill", { title: "Bug report" });

// Agent CI
const { data: runs } = await client.listPipelineRuns(1, 0, 20);
const { data: gates } = await client.evaluateGates(1, { trigger: "publish", versionId: 5 });

// Review & promotion mutations
await client.unwrap(client.approveReview(1, { comment: "LGTM" }));
await client.unwrap(client.rejectPromotion(2, { comment: "Needs work" }));
await client.unwrap(client.withdrawReview(3));
```

## Tool API

```typescript
const { data: resolved } = await client.toolResolve("ns", "skill", "1.0.0");
const { data: install } = await client.toolInstall("ns", "skill");
const { data: diff } = await client.toolDiff("ns", "skill", "1.0.0", "2.0.0");
```

## Maintenance status

The SDK is internally modularized by domain under `src/domains/` and `src/types/`, while `src/index.ts` remains the public barrel. Consumers should continue importing from `@miqro/skillhub-client`; internal module paths are not part of the public compatibility contract.

## Building

```bash
npm install
npm run build
npm test
```

The `dist/` directory is not committed. Build it before referencing locally.

See `guides/typescript-sdk.md` for the full method reference.
