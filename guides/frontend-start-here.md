# Frontend Start Here

This is the first document a frontend engineer should read before building the SkillHub web app.

The backend is ready enough to build a real frontend. The main gap is not backend capability; it is onboarding clarity. Use this guide as the practical starting path, then follow the deeper guides linked below.

## Read This In Order

1. `guides/frontend-start-here.md` — first-week frontend plan and integration rules.
2. `guides/frontend-information-architecture.md` — routes, page hierarchy, app shell, page-to-SDK matrix.
3. `guides/frontend-integration.md` — exact endpoint/SDK method per page, loading states, error states.
4. `guides/typescript-sdk.md` — SDK constructor, auth, unwrap, pagination iterators.
5. `guides/api-usage.md` — raw HTTP reference when the SDK does not expose a helper yet.

Use fixtures from:

```text
server/internal/http/frontend/testdata/contracts/
```

They are committed example response envelopes for frontend read-model pages and are useful for mocks, Storybook stories, and layout work.

## Recommended Stack

Use:

- Vue 3
- Vite
- TypeScript
- Pinia or TanStack Query/Vue Query for client state
- Vue Router

The backend does not require Vue specifically, but Vue 3 + Vite + TypeScript is the recommended default because the project already ships a TypeScript SDK and the planned frontend architecture assumes a modern SPA.

## Local Backend Setup

For local UI development, run the backend with CORS enabled for the frontend origin:

```powershell
cd server
$env:SKILLHUB_DATABASE_URL = "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"
$env:SKILLHUB_CORS_ALLOWED_ORIGINS = "http://localhost:5173"
$env:SKILLHUB_STORAGE_PROVIDER = "local"
$env:SKILLHUB_STORAGE_ROOT = "./data/storage"
$env:SKILLHUB_LOCAL_MODE = "true"
go run ./cmd/skillhub-server
```

For the full local infrastructure stack:

```powershell
docker compose up -d postgres redis minio
docker compose --profile full up -d
```

Docker is optional for frontend work if another developer already provides a running backend.

## Add The SDK

The TypeScript SDK is local to this repository and is not published to npm yet.

In the frontend app:

```json
{
  "dependencies": {
    "@miqro/skillhub-client": "file:../clients/typescript/skillhub"
  }
}
```

Initialize it once and provide it through app context:

```ts
import { SkillHubClient } from "@miqro/skillhub-client";

export const skillhub = new SkillHubClient({
  baseUrl: import.meta.env.VITE_SKILLHUB_API_URL ?? "http://localhost:8080",
  credentials: "include",
});
```

Use `credentials: "include"` for browser session cookies. Bearer-token mode is also supported, but the default frontend should use cookie sessions unless the product decision changes.

## First Week Build Sequence

Build in this order:

1. App shell and router
   - Global header, search box, navigation.
   - Error boundary and empty-state components.
   - SDK provider.

2. Auth
   - Login page with `client.login(username, password)`.
   - Auth bootstrap with `client.me()`.
   - Route guards for 401/403.
   - Logout via raw `POST /api/v1/auth/logout` until the SDK exposes `logout()`.

3. Discover and search
   - `/` and `/search`.
   - Use `client.frontendSearch(...)`.
   - Render filters, sort, pagination, and `availableActions.canCreateSkill`.

4. Skill inspection
   - `/skills/:namespace/:slug`.
   - `/skills/:namespace/:slug/versions/:version`.
   - Use frontend read models, not raw domain endpoints.

5. Namespace pages
   - `/namespaces`.
   - `/namespaces/:namespace`.
   - Render members and actions from read models.

6. Releases and CI
   - Release list/detail.
   - CI run/check/artifact drill-down.
   - Gate evaluation display.

7. Community
   - Issues, discussions, wiki, proposals.
   - Mutation routes exist through the SDK; refetch the page read model after mutation.

8. Review, promotion, governance, admin
   - Review/promotion queue and detail.
   - Approve/reject/withdraw through SDK portal methods.
   - Governance and admin workbenches from frontend read models.

## Core Rules

1. Prefer `SkillHubClient` over raw `fetch`.

2. Use `client.unwrap(...)` in page loaders:

```ts
const page = await client.unwrap(client.frontendSkillDetail(namespace, slug));
```

3. Use envelope mode for inline form errors:

```ts
const result = await client.createIssue(namespace, slug, input);
if (!result.success) {
  showError(result.error.message);
}
```

4. Never derive permissions from role strings in the frontend.

Use `availableActions` returned by the backend:

```ts
if (page.availableActions.canApprove) {
  showApproveButton();
}
```

5. After every mutation, refetch the relevant read model.

6. Do not prefetch unbounded data.

Use explicit `page` and `size` for visible pagination. SDK iterators are for bounded background prefetch only.

7. Treat frontend read-model endpoints as the page contract.

Raw portal/tool endpoints are for mutations, downloads, CI details, or cases where no frontend read model exists.

## Auth And CORS

The backend must allow the frontend origin:

```text
SKILLHUB_CORS_ALLOWED_ORIGINS=http://localhost:5173
```

Cookie sessions require:

- frontend uses `credentials: "include"`;
- backend CORS origin is explicit, not wildcard;
- production uses secure cookies with Redis-backed sessions.

Logout is currently available as an HTTP route but not yet as a public SDK helper:

```ts
await fetch(`${apiBase}/api/v1/auth/logout`, {
  method: "POST",
  credentials: "include",
});
```

After logout, clear frontend auth state and refetch `me()` or redirect to `/login`.

## Error Handling

Use these defaults:

| Status | Frontend behavior |
|---|---|
| 400 | Show validation error near the form/action |
| 401 | Redirect to `/login`, preserving intended destination |
| 403 | Show "You do not have permission" |
| 404 | Show not-found page |
| 409 | Show conflict reason, especially gate failures |
| 429 | Show rate-limit message and retry affordance |
| 503 | Show backend dependency/unavailable message with retry |

## What Not To Build First

Do not start with:

- a marketing landing page;
- git repository concepts;
- custom authorization logic;
- raw HTTP wrappers around every endpoint;
- unbounded queue prefetch;
- a frontend-only permission system.

The first screen should be a usable discovery/search experience.

## Current Backend Gaps To Know

These are known integration notes, not blockers for the frontend:

- `web/` is intentionally empty; the frontend app still needs to be scaffolded.
- The TS SDK is a local package, not published to npm.
- `logout()` is not exposed as an SDK method yet; call the HTTP route directly for now.
- Step logs are not persisted because the CI `LogStore` is not wired.
- Kubernetes manifests and production secret-management docs are not included.

Everything needed for core pages is present through the frontend read-model API, portal mutations, Tool API, OpenAPI spec, TypeScript SDK, and contract fixtures.

## Done Criteria For Frontend MVP

The frontend MVP should prove:

- users can log in and see their auth state;
- users can search and inspect skills;
- users can view namespaces and skill details;
- users can view versions, releases, CI status, and install/download actions;
- community pages render real issue/discussion/wiki/proposal data;
- reviewers can use review and promotion queues;
- admin/governance pages render role-scoped read models;
- all action buttons come from `availableActions`;
- all mutation flows refetch read models after success.

