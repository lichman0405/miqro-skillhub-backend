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

## Building

```bash
npm install
npm run build
npm test
```
