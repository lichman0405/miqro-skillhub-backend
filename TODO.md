# SkillHub — Backend Roadmap

Follow-up plan from the backend evaluation review (as of `master` @ phase 28,
commit `7fd139c`). Items are ordered by priority; items marked "depends on"
should not start until their dependency is done.

## 1. Phase 29 — Runtime Deployment Hardening

- Change the Dockerfile to run as a non-root user.
- Ensure runtime directory permissions are correct, especially the local
  storage root, temp directories, and certificate access.
- Ensure Compose files still allow writable mounted volumes under a
  non-root user.
- Add container/config-level build verification.
- Add a short doc confirming the current production runtime wiring
  (Redis sessions, Redis rate limiting, S3/local storage factory) so future
  reviews don't re-flag already-fixed gaps as open issues.
- Optional: add a `/readyz` endpoint or startup self-check that
  distinguishes liveness from dependency readiness.

## 2. Auditing SDK-vs-HTTP capability gaps

*(depends on: Phase 29)*

Do a full scan across all domains (review, promotion, release, community,
etc.) and list every capability that exists in the SDK but has no exposed
HTTP route. `guides/end-to-end-flow.md` already documents one known gap
(review approve/reject has SDK support but no HTTP handler yet). Fix all
found gaps in one pass instead of discovering and patching them one at a
time.

## 3. Improving test coverage on public attack surface

Prioritize `internal/http/toolapi` (~0.8% coverage) and
`internal/http/portal` (~18.3% coverage) — these are the CLI protocol and
public API surfaces with the largest external attack exposure. Do this
before adding new features.

## 4. Deciding agentrunner LLM integration boundary

Define an `Evaluator` interface boundary for `agentrunner`: the Go side
should only handle orchestration (scheduling, timeouts, retries, result
persistence). The actual LLM-based evaluation logic should sit behind that
interface, implementable either as an external microservice (e.g. Python)
or a direct HTTP call to a model API. Write this decision down as an ADR
before `agentrunner` grows into a mixed-responsibility package.

## 5. Verifying TypeScript SDK modularization progress

Confirm whether `master`'s `phase 25: typescript sdk modularization` has
actually split the single 1947-line `index.ts` into reasonable modules.
This is prep work for freezing the frontend integration contract before a
frontend team starts consuming the SDK.

## 6. Adding multi-instance horizontal scaling integration test

*(depends on: Phase 29)*

Build a docker-compose integration test with 2 server instances + 1 Redis
+ 1 Postgres to verify that session store and distributed rate limiting
state are actually shared across replicas — not just wired in code but
never tested under multiple instances.

## 7. Defining API versioning and deprecation policy

Define an explicit versioning and deprecation process for `/api/v1/*` and
other published routes, ahead of opening the API to third parties or a
frontend team.

## 8. Evaluating audit log immutability and export

Assess whether the `audit` package needs append-only/immutability
guarantees and external SIEM export capability, for enterprise compliance
requirements.
