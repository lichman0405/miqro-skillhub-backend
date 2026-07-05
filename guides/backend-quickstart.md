# Backend Quickstart

How to run the SkillHub backend for local development.

## Environment requirements

- **Go** 1.25+
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
  SKILLHUB_STORAGE_PROVIDER=local \
  SKILLHUB_STORAGE_ROOT=./data/storage \
  go run ./cmd/skillhub-server

# Start the worker (in another terminal)
SKILLHUB_DATABASE_URL="postgres://skillhub:skillhub@localhost:5432/skillhub?sslmode=disable" \
  SKILLHUB_STORAGE_PROVIDER=local \
  SKILLHUB_STORAGE_ROOT=./data/storage \
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
$env:SKILLHUB_STORAGE_PROVIDER = "local"
$env:SKILLHUB_STORAGE_ROOT = "./data/storage"
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
| `SKILLHUB_REDIS_URL` | `redis://localhost:6379/0` | Redis URL (reserved for future adapters; not currently consumed for sessions or rate limiting) |
| `SKILLHUB_STORAGE_PROVIDER` | `local` | Object storage backend: `local` (filesystem) or `s3` (S3-compatible / MinIO) |
| `SKILLHUB_STORAGE_ROOT` | `./data/storage` | Local filesystem storage root (used when provider=local). Falls back to `STORAGE_ROOT` for backward compatibility. |
| `SKILLHUB_STORAGE_ENDPOINT` | `localhost:9000` | S3-compatible endpoint (used when provider=s3) |
| `SKILLHUB_STORAGE_BUCKET` | `skillhub` | Object storage bucket (used when provider=s3) |
| `SKILLHUB_STORAGE_ACCESS_KEY` | `minioadmin` | Object storage access key (used when provider=s3) |
| `SKILLHUB_STORAGE_SECRET_KEY` | `minioadmin` | Object storage secret key (used when provider=s3) |
| `SKILLHUB_STORAGE_USE_SSL` | `false` | Enable TLS for S3 endpoint (used when provider=s3) |
| `SKILLHUB_STORAGE_REGION` | `us-east-1` | S3 region (used when provider=s3) |
| `SKILLHUB_ALLOW_LOCAL_STORAGE_IN_PRODUCTION` | `false` | Allow local storage when `SKILLHUB_LOCAL_MODE=false` (emergency override only) |
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
- Set `SKILLHUB_STORAGE_PROVIDER=s3` and configure endpoint, bucket, and credentials for production. Local filesystem storage is rejected in production mode unless `SKILLHUB_ALLOW_LOCAL_STORAGE_IN_PRODUCTION=true` is explicitly set.
- Object storage must be shared across all server instances for multi-instance deployments.
- Built-in session and rate limiting are in-process (not distributed). Redis-backed sessions and distributed rate limiting are not implemented; `SKILLHUB_REDIS_URL` is reserved for future adapters. Multi-instance deployments should use sticky sessions at the load balancer, external API gateway / load balancer rate limiting, or wait for future Redis-backed adapters.
- Run migrations as an explicit rollout step before starting upgraded servers.
- Back up PostgreSQL and object storage before migrations or schema changes.

### Compose differences

- **`docker-compose.yml`** (dev): uses `SKILLHUB_LOCAL_MODE=true` and `minioadmin` credentials — safe for local development. The `--profile full` server is a local dev profile, not a production configuration.
- **`compose.release.yml`** (release example): uses `SKILLHUB_LOCAL_MODE=false` and requires production credentials as environment variables (`SKILLHUB_DATABASE_URL`, `SKILLHUB_STORAGE_ACCESS_KEY`, `SKILLHUB_STORAGE_SECRET_KEY`). The Compose will refuse to start if they are missing. Supply them via `export` or a `.env` file before running.

## Running without Redis

Redis is reserved for future adapters (distributed sessions, rate limiting). The server currently does **not** consume Redis at runtime. `SKILLHUB_REDIS_URL` is accepted as configuration but is not used.

- Session-based auth uses in-process storage (not persisted across restarts).
- Rate limiting is in-process (bounded: 10000 max buckets, 15min TTL).
- API tokens always work regardless of Redis availability.

## Running without MinIO / S3

Set `SKILLHUB_STORAGE_PROVIDER=local` for local development. The server uses **local filesystem storage** at the path specified by `SKILLHUB_STORAGE_ROOT` (default `./data/storage`, also reads legacy `STORAGE_ROOT` as a fallback). No MinIO or S3 configuration is needed.

For production, set `SKILLHUB_STORAGE_PROVIDER=s3` and configure endpoint, bucket, and credentials. Production mode rejects local storage unless `SKILLHUB_ALLOW_LOCAL_STORAGE_IN_PRODUCTION=true` is explicitly set. Local filesystem storage is unsafe for multi-instance deployments because each instance sees a different filesystem.

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

The local storage adapter creates the directory automatically. If you see permission errors, check that the process has write access to the `SKILLHUB_STORAGE_ROOT` directory.

### "worker can't read package file content"

The worker needs `SKILLHUB_STORAGE_ROOT` to be set to the same directory the server uses for uploads. In development, this is typically `./data/storage` relative to the project root.

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
