# Backend Quickstart

How to run the SkillHub backend for local development.

## Environment requirements

- **Go** 1.24+
- **PostgreSQL** 16+ (with the `skillhub` database created)
- **Make** (optional — all commands can be run without it)
- **Docker** (optional — only needed for Redis, MinIO, or if you want containerized PostgreSQL)

## Quick start (with Docker)

```bash
# Start infrastructure services (PostgreSQL, Redis, MinIO)
docker compose up -d postgres redis minio

# Run database migrations
cd server
SKILLHUB_DATABASE_URL="postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable" \
  go run ./cmd/skillhub-migrate

# Start the server
SKILLHUB_DATABASE_URL="postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable" \
  SKILLHUB_CORS_ALLOWED_ORIGINS="http://localhost:5173" \
  STORAGE_ROOT=./data/storage \
  go run ./cmd/skillhub-server

# Start the worker (in another terminal)
SKILLHUB_DATABASE_URL="postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable" \
  STORAGE_ROOT=./data/storage \
  go run ./cmd/skillhub-worker
```

## Quick start (Windows, no Docker)

### 1. Install PostgreSQL

Download and install PostgreSQL 16 from https://www.postgresql.org/download/windows/.

Create the database:

```sql
CREATE USER skillhub WITH PASSWORD 'skillhub';
CREATE DATABASE skillhub OWNER skillhub;
```

### 2. Set environment variables

```powershell
$env:SKILLHUB_DATABASE_URL = "postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable"
$env:SKILLHUB_CORS_ALLOWED_ORIGINS = "http://localhost:5173"
$env:STORAGE_ROOT = "./data/storage"
```

### 3. Run migrations

```bash
cd server
go run ./cmd/skillhub-migrate
```

### 4. Start the server

```bash
go run ./cmd/skillhub-server
```

The server starts on `http://localhost:8080`.

### 5. Start the worker (optional)

```bash
# In another terminal, cd server first
go run ./cmd/skillhub-worker
```

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `SKILLHUB_DATABASE_URL` | `postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable` | PostgreSQL connection string |
| `SKILLHUB_API_ADDR` | `:8080` | HTTP listen address |
| `SKILLHUB_CORS_ALLOWED_ORIGINS` | empty | Comma-separated browser origins allowed to call the API |
| `REDIS_URL` | (optional) | Redis URL for sessions and rate limiting |
| `STORAGE_ROOT` | `./data/storage` | Local filesystem storage root for package files |
| `SKILLHUB_LOCAL_MODE` | `true` | When `true`, the server auto-creates a local admin user and uses permissive auth. Production must set `false`. |
| `SKILLHUB_TRUSTED_PROXY_CIDRS` | empty | Comma-separated CIDR blocks (e.g. `10.0.0.0/8`) for reverse proxies whose `X-Forwarded-For` header should be trusted. Leave empty to never trust `X-Forwarded-For`. |
| `AGENTCI_LLM_BASE_URL` | (optional) | LLM API base URL for agent CI LLM checks |
| `AGENTCI_LLM_API_KEY` | (optional) | LLM API key |
| `AGENTCI_LLM_MODEL` | (optional) | LLM model name |
| `AGENTCI_LLM_PROVIDER` | (optional) | LLM provider name |
| `AGENTCI_POLL_INTERVAL` | `30s` | Worker polling interval for pending CI runs |

## PostgreSQL requirements

- PostgreSQL 16 or later
- The database must exist before running migrations
- Migration SQL files are in `server/migrations/`
- The migrate command auto-creates all tables (50+ tables across 8 migration groups)
- Seed data includes default platform roles and the global namespace

## Production notes

The quickstart defaults are for local development only. For production, see the [Production readiness](../README.md#production-readiness) checklist in the README. Key points:

- Set `SKILLHUB_LOCAL_MODE=false`; startup will reject known weak defaults.
- Replace all default credentials (PostgreSQL, object storage).
- Set explicit `SKILLHUB_CORS_ALLOWED_ORIGINS` and `SKILLHUB_TRUSTED_PROXY_CIDRS`.
- Use Redis for session/rate-limit behavior when running multiple server instances.
- Run migrations as an explicit rollout step before starting upgraded servers.
- Back up PostgreSQL and object storage before migrations or schema changes.

## Running without Redis

The server works without Redis. When Redis is not configured:
- Session-based auth still works (in-memory, not persisted across restarts)
- Rate limiting is disabled
- API tokens always work

## Running without MinIO / S3

By default, the server uses **local filesystem storage** at the path specified by `STORAGE_ROOT` (default `./data/storage`). No MinIO or S3 configuration is needed for development.

## Common issues

### "cannot connect to database" / routes return 503

The server starts even without a database connection, but all API routes return `503 Service Unavailable`. Make sure:
- PostgreSQL is running
- The `SKILLHUB_DATABASE_URL` environment variable is set correctly
- The `skillhub` database exists

### "Docker is not available"

This is fine. The server does not require Docker. You only need:
- PostgreSQL running natively
- No Redis (the server works without it)
- Filesystem storage (no MinIO needed)

### "make: command not found"

All `make` targets have equivalent direct commands. `make test` = `go test ./...`. See the table above for env var equivalents.

### "Windows path issues"

- Use forward slashes `/` in `STORAGE_ROOT` (e.g., `./data/storage`)
- In PowerShell, use `$env:VAR = "value"` to set env vars
- Avoid spaces in paths; if unavoidable, quote them

### "storage root not writable"

The local storage adapter creates the directory automatically. If you see permission errors, check that the process has write access to the `STORAGE_ROOT` directory.

### "worker can't read package file content"

The worker needs `STORAGE_ROOT` to be set to the same directory the server uses for uploads. In development, this is typically `./data/storage` relative to the project root.

## Binary locations

| Binary | Path | Purpose |
|---|---|---|
| `skillhub-server` | `server/cmd/skillhub-server/` | HTTP API server |
| `skillhub-worker` | `server/cmd/skillhub-worker/` | Background CI worker |
| `skillhub-migrate` | `server/cmd/skillhub-migrate/` | Database migration runner |

## Building

```bash
cd server
go build ./cmd/skillhub-server
go build ./cmd/skillhub-worker
go build ./cmd/skillhub-migrate
```

## Testing

```bash
cd server
go test ./...        # all tests
go vet ./...         # static analysis
go test ./openapi/ -v  # OpenAPI spec validation
```

### PostgreSQL integration tests

Integration tests under `server/tests/integration/` require a real PostgreSQL database.

**Connection priority:** The test helper (`server/internal/testutil/postgres/db.go`) reads only `SKILLHUB_TEST_DATABASE_URL`. If that variable is not set, it falls back to the hard-coded default `postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable` (the same local `skillhub` database used by the dev server). It does **not** read `SKILLHUB_DATABASE_URL`.

**Recommendation:** Always explicitly set `SKILLHUB_TEST_DATABASE_URL` to point at a dedicated test database (e.g. `skillhub_test`) rather than relying on the fallback default:

```powershell
$env:SKILLHUB_TEST_DATABASE_URL="postgres://skillhub:skillhub@localhost:5432/skillhub_test?sslmode=disable"
cd server
go test ./tests/integration/... -v
```

**Important:** Integration tests reset the target schema (`DROP TABLE … CASCADE` + re-migrate) on every run. Never point `SKILLHUB_TEST_DATABASE_URL` (or rely on the fallback default) against a database that contains data you care about. A dedicated test database (e.g. `skillhub_test`) is strongly recommended.

If PostgreSQL is not available, integration tests skip with a clear message. CI environments should run integration tests against a real PostgreSQL instance.

For the TypeScript SDK:

```bash
cd clients/typescript/skillhub
npm install
npm run build        # compile TypeScript
npm test             # run tests
```
