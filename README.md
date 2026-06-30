# Miqro-SkillHub

Go + Vue 3 复现 SkillHub——企业自托管 Agent 技能注册中心。

## Architecture

The backend is **SDK-first**: core behavior lives in public Go SDK packages under `server/sdk/skillhub`. The server binary is a process/HTTP adapter that wires SDK services. No product workflow beyond health/config runs in this phase.

```
miqro-skillhub/
├── server/
│   ├── sdk/skillhub/       # Public Go SDK (importable by other Go programs)
│   │   ├── auth/           # Auth, sessions, API tokens, scopes, RBAC
│   │   ├── namespace/      # Namespace lifecycle, members, policies
│   │   ├── skill/          # Skill publish, query, download, lifecycle
│   │   ├── packagekit/     # Package validation, SKILL.md parsing
│   │   ├── review/         # Review submission, approval, rejection
│   │   ├── promotion/      # Global namespace promotion
│   │   ├── search/         # Search query, indexing, visibility scope
│   │   ├── storage/        # Object storage interface
│   │   ├── eventbus/       # Domain event bus interface
│   │   ├── errors/         # Typed error model
│   │   ├── uow/            # Unit-of-work / transaction boundary
│   │   └── ...             # label, social, report, governance, notification, security, audit
│   ├── internal/
│   │   ├── config/         # Environment configuration
│   │   ├── http/           # HTTP routes (health only in Phase 01)
│   │   ├── adapters/
│   │   │   └── postgres/   # PostgreSQL repository implementations
│   │   └── testutil/
│   │       └── postgres/   # Integration test helpers
│   ├── migrations/         # PostgreSQL migration SQL files
│   │   ├── 001_users_auth.sql
│   │   ├── 002_namespaces.sql
│   │   ├── 003_skills.sql
│   │   ├── 004_review_governance.sql
│   │   └── 005_labels_social_search.sql
│   ├── tests/
│   │   └── integration/
│   │       └── repository/ # Repository integration tests
│   ├── cmd/
│   │   ├── skillhub-server/  # HTTP server entry point
│   │   ├── skillhub-migrate/ # Database migrations (placeholder)
│   │   └── skillhub-worker/  # Background workers (placeholder)
│   └── tests/
├── web/                    # Vue 3 frontend
├── docs/                   # Architecture and phase documentation
├── docker-compose.yml
└── README.md
```

## Phase 01 — SDK Foundation

Status: ✅ Complete

- Go module skeleton at `server/go.mod` (module `miqro-skillhub/server`)
- SDK root `Service` struct with typed fields for all domain services
- Typed error model (`bad_request`, `forbidden`, `not_found`, `conflict`, `unauthorized`, `internal`)
- Event bus interface (`Publish`) with synchronous no-op adapter
- Unit-of-work `Transactor` interface with no-op adapter
- Object storage `Store` interface matching source `ObjectStorageService`
- Package docs for auth, namespace, skill, packagekit, review, promotion, search, label, social, report, governance, notification, security, audit
- Environment config loader (API addr, database URL, Redis URL, storage, local mode)
- Health routes: `GET /healthz`, `GET /readyz`
- `cmd/skillhub-server`: HTTP server wiring config and health routes
- `cmd/skillhub-migrate`: placeholder (migrations start in Phase 02)
- `cmd/skillhub-worker`: placeholder (workers start in later phases)
- Docker Compose: PostgreSQL 16, Redis 7, MinIO, server (profile: full)

## Phase 02 — Schema and Repositories

Status: ✅ Complete

- 35 PostgreSQL tables across 5 migration groups matching source Flyway schema
- Migration groups: 001 (users/auth/RBAC), 002 (namespaces), 003 (skills), 004 (review/governance), 005 (labels/social/search/security)
- All timestamps use TIMESTAMPTZ; seed data for roles, permissions, and global namespace
- SDK repository interfaces in auth, namespace, skill, review, label, social, report, governance, security, audit packages
- PostgreSQL adapter implementations under `internal/adapters/postgres` using pgx v5
- Transactor implementation for PostgreSQL transaction boundaries
- Integration test utilities and repository integration tests
- Migration runner with reset support

## Commands

```bash
# Run all Go tests
make test

# Vet and build the server
make test-server

# Run the server locally
make run-server

# Validate docker-compose.yml
make compose-config

# Reset and re-apply database migrations
make db-reset

# Start infrastructure services
docker compose up -d postgres redis minio
```

## 工作流

```
Codex (规划+审查)  ←→  Claude Code (实现)
         ↘              ↗
        你 (审查+决策)
```

1. **Codex** 读取 `docs/codex-input-original-analysis.md`，理解原始项目
2. **Codex** 输出分阶段实现方案，每阶段末尾包含「给 Claude Code 的指令」
3. **你** 审查方案
4. **Claude Code** 逐阶段实现
5. **Codex** 审查实现结果
