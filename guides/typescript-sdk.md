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

// Custom base URL
const client = new SkillHubClient("https://skillhub.example.com");
```

## Setting auth token

The client uses `fetch()` under the hood. To send a Bearer token, store the token and include it in future requests by subclassing or wrapping:

```typescript
class AuthenticatedClient extends SkillHubClient {
  constructor(baseUrl: string, private token: string) {
    super(baseUrl);
  }

  // Override fetch to inject Authorization header — or use a wrapper pattern.
  // For now, the client does not auto-attach tokens; pass credentials: 'include'
  // for session-based auth or attach headers at the application level.
}
```

For session-based auth, call `login()` which sets the session cookie:

```typescript
const { data: user } = await client.login("username", "password");
// Cookie is set; subsequent requests with credentials: 'include' use the session.
```

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
// reviews.tasks, reviews.pendingCount, reviews.availableActions.canReview

// Admin page
const { data: admin } = await client.frontendAdmin();
// admin.stats.totalSkills, admin.availableActions.canManageSkills

// Community pages
const { data: issues } = await client.frontendIssueList("ns", "my-skill");
const { data: discussions } = await client.frontendDiscussionList("ns", "my-skill");
const { data: wiki } = await client.frontendWikiList("ns", "my-skill");
const { data: proposals } = await client.frontendProposalList("ns", "my-skill");
```

## Error handling

The client returns `Envelope<T>` for every call:

```typescript
interface Envelope<T> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
}

// Usage pattern:
const result = await client.search({ keyword: "test" });
if (result.success && result.data) {
  // use result.data
} else {
  console.error(result.error?.code, result.error?.message);
}
```

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

## Building and testing

```bash
cd clients/typescript/skillhub
npm install
npm run build         # tsc
npm test              # node --test dist/**/*.test.js
npx tsc --noEmit      # typecheck without emitting
```
