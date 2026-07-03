# @miqro/skillhub-client

Generated TypeScript client for the SkillHub HTTP API.

## Usage

```typescript
import { SkillHubClient } from "@miqro/skillhub-client";

const client = new SkillHubClient("http://localhost:8080");

// Login
const { data: user } = await client.login("alice", "password123");

// Search skills
const { data: results } = await client.search({ keyword: "agent", installableOnly: true });

// Get skill detail
const { data: skill } = await client.getSkill("my-namespace", "my-skill");
```

## Frontend read-model examples

Every `/api/v1/frontend/*` route has a typed client method. Responses are wrapped in `Envelope<T>` with `success`, `data`, and optional `error` fields. Path parameters are encoded automatically.

```typescript
// Discover page
const { data: home } = await client.frontendSearch({
  keyword: "agent",
  sortBy: "downloads",
  page: 0,
  size: 20,
  labelSlugs: ["go"],
  installableOnly: true,
});

// Skill detail page
const { data: skillPage } = await client.frontendSkillDetail("my-namespace", "my-skill");

// Release detail page
const { data: releasePage } = await client.frontendReleaseDetail("my-namespace", "my-skill", 1);
```

See `guides/typescript-sdk.md` for the full list of frontend methods.

## Building

```bash
npm install
npm run build
npm test
```
