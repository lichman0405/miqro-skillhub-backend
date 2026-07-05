# web/

No frontend application is scaffolded yet. This directory is a placeholder for the future SkillHub web frontend.

## Current State

- **No frontend app exists.** Do not add frontend dependencies in this phase.
- **No framework is installed.** No `package.json`, `node_modules`, or build tooling.

## Recommended Stack

When the frontend is built, the recommended default stack is **Vue 3 + Vite + TypeScript**. This aligns with the original target architecture and the existing TypeScript SDK (`@miqro/skillhub-client`).

## Contract Source

The committed frontend contract source is:

- **TypeScript SDK** at `clients/typescript/skillhub/` — typed client with auth, errors, and pagination.
- **Guides** — `guides/frontend-information-architecture.md` (page plan) and `guides/frontend-integration.md` (endpoint usage).

Do not guess API shapes from raw HTTP calls. Use the SDK methods documented in these guides.

## Getting Started (Future)

When frontend implementation begins:

1. Read `guides/frontend-information-architecture.md` for the route inventory, app shell, and page-to-SDK matrix.
2. Read `guides/frontend-integration.md` for per-page endpoint details and permission button rules.
3. Scaffold a Vue 3 + Vite + TypeScript project in this directory.
4. Add `@miqro/skillhub-client` as a local dependency.
5. Follow the frontend build phases defined in the information architecture guide.

Do not add any frontend dependency, framework, or build tooling until a future phase explicitly authorizes frontend app scaffold.
