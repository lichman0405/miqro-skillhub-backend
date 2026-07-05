# Miqro-SkillHub

Self-hosted Agent Skill Registry — a backend service for publishing, reviewing, releasing, and installing agent skills.

## What is Miqro-SkillHub?

Miqro-SkillHub is an enterprise self-hosted registry for agent skills. It provides:

- **Skill publishing** — upload and validate skill packages (SKILL.md + files)
- **Namespace management** — organize skills by organization/team
- **CI/CD pipeline** — deterministic checks (manifest, secrets, docs) run on every publish
- **Release management** — versioned, gated releases with draft/publish workflow
- **Community features** — issues, discussions, wiki, change proposals per skill
- **Review workflow** — submit skills for review, approve/reject with gate enforcement
- **Search and discovery** — search across public/namespace-scoped skills
- **Tool API** — miqro CLI integration for resolve, install, diff, validate, publish

## Architecture

The backend is **SDK-first**: core behavior lives in public Go SDK packages under `server/sdk/skillhub`. The server binary is a process/HTTP adapter that wires SDK services.

```
miqro-skillhub/
├── server/
│   ├── sdk/skillhub/        # Public Go SDK (importable by other Go programs)
│   │   ├── agentci/         # CI pipeline, checks, gates, worker execution
│   │   ├── auth/            # Auth, sessions, API tokens, scopes, RBAC
│   │   ├── community/       # Issues, discussions, wiki, proposals
│   │   ├── namespace/       # Namespace lifecycle, members, policies
│   │   ├── packagekit/      # Package validation, SKILL.md parsing
│   │   ├── release/         # Release lifecycle, assets, gate enforcement
│   │   ├── review/          # Review submission, approval, gate enforcement
│   │   ├── search/          # Search query, indexing, visibility scope
│   │   ├── skill/           # Skill publish, query, download, lifecycle
│   │   ├── storage/         # Object storage interface
│   │   ├── tooling/         # Tool API (hash, resolve, install, diff)
│   │   ├── eventbus/        # Domain event bus interface
│   │   ├── errors/          # Typed error model
│   │   └── uow/             # Unit-of-work / transaction boundary
│   ├── internal/
│   │   ├── adapters/
│   │   │   ├── postgres/       # PostgreSQL repository implementations
│   │   │   ├── localstorage/   # Local filesystem object storage
│   │   │   ├── s3/             # S3/MinIO object storage adapter
│   │   │   ├── storagefactory/ # Unified storage factory (local vs s3)
│   │   │   └── agentrunner/    # CI runner (local + LLM)
│   │   ├── config/          # Environment configuration
│   │   ├── http/            # HTTP routes and handlers
│   │   │   ├── portal/      # /api/v1/* routes
│   │   │   ├── frontend/    # /api/v1/frontend/* read-model routes
│   │   │   ├── toolapi/     # /api/tool/v1/* routes
│   │   │   ├── cliapi/      # /api/cli/v1/* routes
│   │   │   ├── middleware/  # Auth, rate limiting, error handling
│   │   │   └── observability/ # Logging, metrics
│   │   └── testutil/        # Integration test helpers
│   ├── migrations/          # PostgreSQL migration SQL files (8 groups)
│   ├── openapi/             # OpenAPI 3.0.3 specification
│   ├── cmd/
│   │   ├── skillhub-server/ # HTTP server entry point
│   │   ├── skillhub-worker/ # Background CI worker
│   │   └── skillhub-migrate/# Database migration runner
│   └── go.mod
├── clients/
│   └── typescript/skillhub/ # TypeScript SDK (@miqro/skillhub-client)
├── guides/                  # Integration and usage guides
│   ├── backend-quickstart.md
│   ├── api-usage.md
│   ├── typescript-sdk.md
│   ├── frontend-integration.md
│   ├── frontend-information-architecture.md
│   └── end-to-end-flow.md
├── web/                     # Placeholder for future frontend; see web/README.md
├── docker-compose.yml
└── README.md
```

## Current backend capabilities

| Domain | Status |
|---|---|
| Auth (login, register, tokens, RBAC) | ✅ |
| Namespace CRUD + members | ✅ |
| Skill publish, query, download | ✅ |
| Package validation + manifest | ✅ |
| Search (keyword, filters, pagination) | ✅ |
| CI pipeline (manifest, secrets, docs) | ✅ |
| CI worker (poll + execute) | ✅ |
| CI gate enforcement | ✅ |
| Release lifecycle (draft → publish) | ✅ |
| Review workflow (submit, approve, reject) | ✅ SDK + HTTP adapters + frontend read models |
| Community (issues, discussions, wiki, proposals) | ✅ |
| Frontend read-model routes | ✅ core pages + review/promotion/governance/admin read models wired with real data |
| Tool API (miqro CLI protocol) | ✅ |
| OpenAPI 3.0.3 spec | ✅ |
| TypeScript SDK | ✅ |
| PostgreSQL migrations (8 groups, ~50 tables) | ✅ |
| Docker Compose stack | ✅ |

## Local development

See **[guides/backend-quickstart.md](guides/backend-quickstart.md)** for step-by-step setup.

### Quick start (with Docker)

```bash
docker compose up -d postgres
cd server
SKILLHUB_DATABASE_URL="postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable" go run ./cmd/skillhub-migrate
SKILLHUB_DATABASE_URL="postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable" SKILLHUB_CORS_ALLOWED_ORIGINS="http://localhost:5173" SKILLHUB_STORAGE_PROVIDER=local SKILLHUB_STORAGE_ROOT=./data/storage go run ./cmd/skillhub-server
```

The `--profile full` server in `docker-compose.yml` uses `SKILLHUB_LOCAL_MODE=true` and `minioadmin` credentials — it is a local development profile, not a production configuration. A `minio-init` helper container automatically creates the `skillhub` bucket before the server starts, so the full profile works with a single `docker compose --profile full up -d`. For a production-style Compose, see `compose.release.yml`.

### Quick start (Windows, no Docker)

Install PostgreSQL 16, create the `skillhub` database, then:

```powershell
$env:SKILLHUB_DATABASE_URL = "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"
$env:SKILLHUB_CORS_ALLOWED_ORIGINS = "http://localhost:5173"
$env:SKILLHUB_STORAGE_PROVIDER = "local"
$env:SKILLHUB_STORAGE_ROOT = "./data/storage"
cd server
go run ./cmd/skillhub-migrate
go run ./cmd/skillhub-server
```

### Commands

```bash
# Go tests (all packages)
cd server && go test ./...

# Static analysis
cd server && go vet ./...

# Build binaries
cd server && go build ./cmd/skillhub-server
cd server && go build ./cmd/skillhub-worker
cd server && go build ./cmd/skillhub-migrate

# OpenAPI spec validation
cd server && go test ./openapi/ -v

# TypeScript SDK
cd clients/typescript/skillhub && npm install && npm run build && npm test
```

## Binary locations

| Binary | Path | Purpose |
|---|---|---|
| `skillhub-server` | `server/cmd/skillhub-server/` | HTTP API server |
| `skillhub-worker` | `server/cmd/skillhub-worker/` | Background CI worker |
| `skillhub-migrate` | `server/cmd/skillhub-migrate/` | Database migration runner |

## Production readiness

Before running outside local development:

- Set `SKILLHUB_LOCAL_MODE=false`. Startup rejects known local defaults (`minioadmin` credentials, localhost database URL) in this mode.
- Set `SKILLHUB_STORAGE_PROVIDER=s3` for production. Local filesystem storage is allowed in production only with the explicit emergency override `SKILLHUB_ALLOW_LOCAL_STORAGE_IN_PRODUCTION=true`. Without this override, production mode rejects local storage so multi-instance deployments are not silently broken.
- Replace the default PostgreSQL URL and credentials; do not use `skillhub:skillhub`.
- Replace object-storage credentials; do not use `minioadmin:minioadmin`.
- Set explicit `SKILLHUB_CORS_ALLOWED_ORIGINS` for browser clients. Avoid wildcard origins for credentialed requests.
- Set `SKILLHUB_TRUSTED_PROXY_CIDRS` only to real reverse proxy / load-balancer CIDRs. Leave empty if no trusted proxy exists — `X-Forwarded-For` is never trusted when empty, so spoofed headers cannot bypass rate limiting.
- **Object storage must be shared across all server instances.** Local filesystem storage is unsafe for multi-instance deployments because each instance sees a different filesystem. Use S3/MinIO so every instance reads/writes the same package files, release assets, and CI artifacts.
- The server's built-in session and rate-limit implementations are in-process (not distributed). For multi-instance production deployments, use sticky sessions at your load balancer, external API gateway / load balancer rate limiting, or wait for future Redis-backed adapters. The current in-process rate limiter is bounded (10000 max buckets, 15min TTL).
- **Redis-backed sessions and distributed rate limiting are NOT implemented.** `SKILLHUB_REDIS_URL` is reserved for future adapters. The server does not consume Redis at runtime.
- Run database migrations as an explicit rollout step before starting upgraded servers.
- Back up PostgreSQL and object storage before migrations or data-model upgrades.
- Scrape `/metrics` with Prometheus or equivalent monitoring (see `monitoring/prometheus.yml` for a starter config).
- See `compose.release.yml` for a release-style Compose example. It requires production credentials as environment variables; the Compose will refuse to start if `SKILLHUB_DATABASE_URL`, `SKILLHUB_STORAGE_ACCESS_KEY`, or `SKILLHUB_STORAGE_SECRET_KEY` are not set. MinIO and the server share the same `SKILLHUB_STORAGE_ACCESS_KEY` / `SKILLHUB_STORAGE_SECRET_KEY` — both must use the same production credentials (not `minioadmin`). Supply them on the command line or via a `.env` file.

## Coverage

Cross-package coverage (more realistic than per-package default):

```bash
cd server
go test -coverpkg=./... -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

The `-coverpkg=./...` flag ensures coverage is measured across all packages, even when tests in one package exercise adapters or HTTP handlers in another.

## CI verification

`.github/workflows/pr-tests.yml` runs on every PR against `main`/`master`:

- `go vet ./...` — static analysis
- `go test -race -count=1 ./...` — all tests with race detector and a real PostgreSQL service
- Builds `skillhub-server`, `skillhub-migrate`, and `skillhub-worker`
- Builds the server Docker image

CI does **not** currently verify:
- TypeScript SDK build/tests
- End-to-end or integration tests against external object storage (MinIO/S3)
- Docker Compose stack or `compose.release.yml` configuration
- Production deployment, Kubernetes, or npm publishing

OpenAPI spec validation is covered by `go test ./...` (which includes the `server/openapi` package). The explicit `go test ./openapi/ -v` command remains useful for local debugging.

Local Docker and make availability are optional; direct Go commands are the baseline verification path.

## OpenAPI

The OpenAPI 3.0.3 specification is at `server/openapi/openapi.yaml`. It documents all portal, tool, CLI, frontend, release, community, and agent CI routes.

Validate: `cd server && go test ./openapi/ -v`

## TypeScript SDK

The TypeScript client is at `clients/typescript/skillhub/`.

```typescript
import { SkillHubClient } from "@miqro/skillhub-client";

// Options constructor with auth, custom fetch, headers
const client = new SkillHubClient({
  baseUrl: "http://localhost:8080",
  credentials: "include",
  token: "sk_...",
});

// Envelope mode (backward compatible)
const { data } = await client.search({ keyword: "agent" });

// Unwrap mode (typed errors)
const results = await client.unwrap(client.search({ keyword: "agent" }));

// Bounded pagination iterators
for await (const page of client.iterFrontendReviews({ size: 50, maxPages: 5 })) {
  console.log(page.tasks);
}
```

See **[guides/typescript-sdk.md](guides/typescript-sdk.md)** for full usage.

## Documentation

| Guide | Description |
|---|---|
| [guides/backend-quickstart.md](guides/backend-quickstart.md) | Environment setup, env vars, Docker/Windows notes |
| [guides/api-usage.md](guides/api-usage.md) | API reference with request/response examples |
| [guides/typescript-sdk.md](guides/typescript-sdk.md) | TS SDK installation and usage |
| [guides/frontend-integration.md](guides/frontend-integration.md) | Per-page API calls, permission buttons, empty/error states |
| [guides/frontend-information-architecture.md](guides/frontend-information-architecture.md) | Frontend IA: routes, app shell, page-to-SDK matrix, build phases |
| [guides/end-to-end-flow.md](guides/end-to-end-flow.md) | Complete happy path from upload to release |
