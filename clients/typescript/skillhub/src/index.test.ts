/**
 * Tests for the SkillHub TypeScript client.
 *
 * These tests validate the client's type safety, frontend contract
 * coverage, URL construction, auth configuration, error handling,
 * and pagination iterator behavior.
 */
import { describe, it, before, after } from "node:test";
import assert from "node:assert/strict";
import {
  SkillHubClient,
  SkillHubError,
  type SkillHubClientOptions,
  type Envelope,
  type Principal,
  type SearchResult,
  type SkillDetail,
  type RegistrySearchReadModel,
  type RegistrySearchActions,
  type SkillDetailReadModel,
  type SkillDetailActions,
  type VersionDetailReadModel,
  type VersionActions,
  type PublishValidateReadModel,
  type PublishValidateActions,
  type NamespaceListReadModel,
  type NamespaceListActions,
  type NamespaceDetailReadModel,
  type NamespaceDetailActions,
  type ReviewQueueReadModel,
  type ReviewQueueActions,
  type ReviewDetailReadModel,
  type ReviewDetailActions,
  type PromotionQueueReadModel,
  type PromotionQueueActions,
  type PromotionDetailReadModel,
  type PromotionDetailActions,
  type GovernanceWorkbenchReadModel,
  type GovernanceWorkbenchActions,
  type AdminPageReadModel,
  type AdminPageActions,
  type WorkspaceMetadataResponse,
  type PackageEntry,
  type PackageManifest,
  type PackageHashResponse,
  type ResolveResult,
  type InstallTarget,
  type AgentRuntime,
  type VersionDiff,
  type DiffSummary,
  type DiffFile,
  type DiffHunk,
  type DiffLine,
  type ToolValidateResponse,
  type ToolPublishResponse,
  type EvaluateRequest,
  type EvaluateResponse,
  type ProposalRequest,
  type ProposalResponse,
  type ReleaseListReadModel,
  type ReleaseDetailReadModel,
  type IssueListReadModel,
  type IssueDetailReadModel,
  type DiscussionListReadModel,
  type DiscussionDetailReadModel,
  type WikiPageListReadModel,
  type WikiPageDetailReadModel,
  type ProposalListReadModel,
  type ProposalDetailReadModel,
  type PipelineRunListResult,
  type PageIteratorOptions,
} from "./index.js";

// ── Helpers ──────────────────────────────────────────────────────────────

/** Create a mock fetch that captures requests and returns canned JSON. */
function mockFetch(
  responseBody: unknown,
  status = 200,
): typeof globalThis.fetch {
  return (async (input: RequestInfo | URL, init?: RequestInit) => {
    return {
      ok: status >= 200 && status < 300,
      status,
      json: async () => responseBody,
      headers: new Headers(),
    } as Response;
  }) as typeof globalThis.fetch;
}

/** Record of a captured fetch call. */
interface CapturedCall {
  url: string;
  init?: RequestInit;
}

/** Create a mock fetch that captures calls and returns canned JSON. */
function capturingFetch(
  responseBody: unknown,
  capture: { calls: CapturedCall[] },
  status = 200,
): typeof globalThis.fetch {
  return (async (input: RequestInfo | URL, init?: RequestInit) => {
    capture.calls.push({
      url: typeof input === "string" ? input : input.toString(),
      init,
    });
    return {
      ok: status >= 200 && status < 300,
      status,
      json: async () => responseBody,
      headers: new Headers(),
    } as Response;
  }) as typeof globalThis.fetch;
}

// ── Constructor tests ────────────────────────────────────────────────────

describe("SkillHubClient constructor", () => {
  it("constructs with default base URL", () => {
    const client = new SkillHubClient();
    assert.ok(client instanceof SkillHubClient);
  });

  it("constructs with custom base URL string", () => {
    const client = new SkillHubClient("https://skillhub.example.com");
    assert.ok(client instanceof SkillHubClient);
  });

  it("constructs with options object", () => {
    const client = new SkillHubClient({ baseUrl: "https://api.example.com" });
    assert.ok(client instanceof SkillHubClient);
  });

  it("constructs with empty options object (defaults)", () => {
    const client = new SkillHubClient({});
    assert.ok(client instanceof SkillHubClient);
  });

  it("strips trailing slash from base URL", async () => {
    const capture = { calls: [] as CapturedCall[] };
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080/",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
    await client.frontendNamespaces();
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/namespaces",
    );
  });
});

// ── Auth configuration tests ─────────────────────────────────────────────

describe("Auth configuration", () => {
  it("sends bearer token via Authorization header", async () => {
    const capture = { calls: [] as CapturedCall[] };
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      token: "sk_test123",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
    await client.me();
    const authHeader = (capture.calls.at(-1)!.init?.headers as Headers).get(
      "Authorization",
    );
    assert.strictEqual(authHeader, "Bearer sk_test123");
  });

  it("sends dynamic token via getToken", async () => {
    const capture = { calls: [] as CapturedCall[] };
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      getToken: () => "dynamic_token_456",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
    await client.frontendGovernance();
    const authHeader = (capture.calls.at(-1)!.init?.headers as Headers).get(
      "Authorization",
    );
    assert.strictEqual(authHeader, "Bearer dynamic_token_456");
  });

  it("getToken returning undefined does not set auth header", async () => {
    const capture = { calls: [] as CapturedCall[] };
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      getToken: () => undefined,
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
    await client.frontendAdmin();
    const headers = capture.calls.at(-1)!.init?.headers as Headers;
    assert.strictEqual(headers.has("Authorization"), false);
  });

  it("passes credentials from options", async () => {
    const capture = { calls: [] as CapturedCall[] };
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      credentials: "include",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
    await client.frontendSearch();
    assert.strictEqual(capture.calls.at(-1)!.init?.credentials, "include");
  });

  it("per-request credentials override constructor default", async () => {
    const capture = { calls: [] as CapturedCall[] };
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      credentials: "include",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
    // me() sets credentials: "include" in init
    await client.me();
    // Should have the per-request value
    assert.strictEqual(capture.calls.at(-1)!.init?.credentials, "include");
  });

  it("custom headers are merged into every request", async () => {
    const capture = { calls: [] as CapturedCall[] };
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      headers: { "X-Client": "web", "X-Version": "1.0" },
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
    await client.frontendNamespaces();
    const headers = capture.calls.at(-1)!.init?.headers as Headers;
    assert.strictEqual(headers.get("X-Client"), "web");
    assert.strictEqual(headers.get("X-Version"), "1.0");
    // Default content-type is still set
    assert.strictEqual(headers.get("Content-Type"), "application/json");
  });

  it("custom fetch is used when provided", async () => {
    let called = false;
    const customFetch = (async () => {
      called = true;
      return {
        ok: true,
        status: 200,
        json: async () => ({ success: true, data: {} }),
        headers: new Headers(),
      } as Response;
    }) as typeof globalThis.fetch;

    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: customFetch,
    });
    await client.frontendGovernance();
    assert.strictEqual(called, true);
  });

  it("token takes precedence over getToken", async () => {
    const capture = { calls: [] as CapturedCall[] };
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      token: "static_token",
      getToken: () => "dynamic_token",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
    await client.me();
    const authHeader = (capture.calls.at(-1)!.init?.headers as Headers).get(
      "Authorization",
    );
    assert.strictEqual(authHeader, "Bearer static_token");
  });
});

// ── URL construction tests: portal methods ───────────────────────────────

describe("Portal client URL construction", () => {
  let capture = { calls: [] as CapturedCall[] };
  let client: SkillHubClient;

  before(() => {
    capture = { calls: [] as CapturedCall[] };
    client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
  });

  it("getSkill encodes namespace and slug", async () => {
    await client.getSkill("team alpha", "my/skill");
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/team%20alpha/my%2Fskill",
    );
  });

  it("getNamespace encodes slug", async () => {
    await client.getNamespace("team alpha");
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/namespaces/team%20alpha",
    );
  });

  it("search builds query params", async () => {
    await client.search({
      keyword: "agent",
      sortBy: "downloads",
      page: 1,
      size: 20,
      labelSlugs: ["go", "ci"],
      installableOnly: true,
    });
    const url = capture.calls.at(-1)!.url;
    assert.ok(url.includes("keyword=agent"));
    assert.ok(url.includes("sortBy=downloads"));
    assert.ok(url.includes("page=1"));
    assert.ok(url.includes("size=20"));
    assert.ok(url.includes("labelSlugs=go%2Cci"));
    assert.ok(url.includes("installableOnly=true"));
  });

  it("search omits undefined params", async () => {
    await client.search({});
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/search",
    );
  });
});

// ── URL construction tests: release methods ──────────────────────────────

describe("Release URL construction", () => {
  let capture = { calls: [] as CapturedCall[] };
  let client: SkillHubClient;

  before(() => {
    capture = { calls: [] as CapturedCall[] };
    client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
  });

  it("listReleases encodes namespace and slug, adds page/size", async () => {
    await client.listReleases("team alpha", "my/skill", 0, 50);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/team%20alpha/my%2Fskill/releases?page=0&size=50",
    );
  });

  it("getLatestRelease encodes namespace and slug", async () => {
    await client.getLatestRelease("ns", "my-skill", "beta");
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/ns/my-skill/releases/latest?channel=beta",
    );
  });

  it("getRelease builds correct path with releases segment", async () => {
    await client.getRelease("ns", "skill", 42);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/ns/skill/releases/42",
    );
  });

  it("createRelease builds correct path", async () => {
    await client.createRelease("ns", "skill", {
      versionId: 5,
      title: "v1.0",
    });
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/ns/skill/releases",
    );
  });

  it("publishRelease builds correct path", async () => {
    await client.publishRelease("ns", "skill", 1);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/ns/skill/releases/1/publish",
    );
  });
});

// ── URL construction tests: tool methods ──────────────────────────────────

describe("Tool API URL construction", () => {
  let capture = { calls: [] as CapturedCall[] };
  let client: SkillHubClient;

  before(() => {
    capture = { calls: [] as CapturedCall[] };
    client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
  });

  it("toolResolve encodes namespace and slug", async () => {
    await client.toolResolve("team alpha", "my/skill", "1.0.0");
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/tool/v1/skills/team%20alpha/my%2Fskill/resolve?version=1.0.0",
    );
  });

  it("toolInstall encodes namespace and slug", async () => {
    await client.toolInstall("team alpha", "my/skill");
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/tool/v1/skills/team%20alpha/my%2Fskill/install",
    );
  });

  it("toolDiff encodes path params", async () => {
    await client.toolDiff("ns", "skill", "1.0.0", "2.0.0");
    const url = capture.calls.at(-1)!.url;
    assert.ok(url.includes("/api/tool/v1/skills/ns/skill/diff"));
    assert.ok(url.includes("from=1.0.0"));
    assert.ok(url.includes("to=2.0.0"));
  });

  it("toolValidate encodes namespace", async () => {
    const blob = new Blob(["test"]);
    await client.toolValidate("team alpha", blob);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/tool/v1/skills/team%20alpha/validate",
    );
  });

  it("toolPublish encodes namespace", async () => {
    const blob = new Blob(["test"]);
    await client.toolPublish("team alpha", blob);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/tool/v1/skills/team%20alpha/publish",
    );
  });
});

// ── URL construction tests: community portal methods ──────────────────────

describe("Community portal URL construction", () => {
  let capture = { calls: [] as CapturedCall[] };
  let client: SkillHubClient;

  before(() => {
    capture = { calls: [] as CapturedCall[] };
    client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
  });

  it("listIssues encodes namespace and slug", async () => {
    await client.listIssues("team alpha", "my/skill", {
      status: "OPEN",
      page: 0,
      size: 20,
    });
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/team%20alpha/my%2Fskill/issues?status=OPEN&page=0&size=20",
    );
  });

  it("getIssue builds correct path with issues segment", async () => {
    await client.getIssue("ns", "skill", 5);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/ns/skill/issues/5",
    );
  });

  it("addIssueComment builds correct path", async () => {
    await client.addIssueComment("ns", "skill", 5, { body: "comment" });
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/ns/skill/issues/5/comments",
    );
  });

  it("getDiscussion builds correct path with discussions segment", async () => {
    await client.getDiscussion("ns", "skill", 3);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/ns/skill/discussions/3",
    );
  });

  it("getWikiPage encodes pageSlug", async () => {
    await client.getWikiPage("ns", "skill", "getting started");
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/ns/skill/wiki/getting%20started",
    );
  });

  it("getProposal builds correct path with proposals segment", async () => {
    await client.getProposal("ns", "skill", 7);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/ns/skill/proposals/7",
    );
  });

  it("communitySearch encodes all params", async () => {
    await client.communitySearch("ns", "skill", {
      query: "bug",
      types: "issues,discussions",
      page: 0,
      size: 10,
    });
    const url = capture.calls.at(-1)!.url;
    assert.ok(url.includes("/community/search"));
    assert.ok(url.includes("query=bug"));
    assert.ok(url.includes("types=issues%2Cdiscussions"));
    assert.ok(url.includes("page=0"));
    assert.ok(url.includes("size=10"));
  });
});

// ── URL construction tests: agent CI methods ──────────────────────────────

describe("Agent CI URL construction", () => {
  let capture = { calls: [] as CapturedCall[] };
  let client: SkillHubClient;

  before(() => {
    capture = { calls: [] as CapturedCall[] };
    client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
  });

  it("listPipelineRuns builds correct path", async () => {
    await client.listPipelineRuns(1, 0, 20);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/1/ci/runs?page=0&size=20",
    );
  });

  it("getPipelineRun builds correct path", async () => {
    await client.getPipelineRun(1, 5);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/1/ci/runs/5",
    );
  });

  it("listCheckRuns builds correct path", async () => {
    await client.listCheckRuns(1, 5);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/skills/1/ci/runs/5/checks",
    );
  });

  it("evaluateGates builds query params", async () => {
    await client.evaluateGates(1, { trigger: "publish", versionId: 3 });
    const url = capture.calls.at(-1)!.url;
    assert.ok(url.includes("/api/v1/skills/1/ci/gates"));
    assert.ok(url.includes("trigger=publish"));
    assert.ok(url.includes("versionId=3"));
  });
});

// ── Frontend client URL construction ──────────────────────────────────────

describe("Frontend client URL construction", () => {
  let capture = { calls: [] as CapturedCall[] };
  let client: SkillHubClient;

  before(() => {
    capture = { calls: [] as CapturedCall[] };
    client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: capturingFetch({ success: true, data: {} }, capture),
    });
  });

  it("frontendSearch builds expected query string", async () => {
    await client.frontendSearch({
      keyword: "agent tools",
      sortBy: "downloads",
      page: 1,
      size: 20,
      labelSlugs: ["go", "ci"],
      installableOnly: true,
    });
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/search?q=agent+tools&sort=downloads&page=1&size=20&labels=go%2Cci&installable=true",
    );
  });

  it("frontendSearch omits empty query params", async () => {
    await client.frontendSearch({});
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/search",
    );
  });

  it("frontendSkillDetail encodes path params", async () => {
    await client.frontendSkillDetail("team alpha", "my/skill");
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/skills/team%20alpha/my%2Fskill",
    );
  });

  it("frontendVersionDetail encodes version", async () => {
    await client.frontendVersionDetail("ns", "skill", "1.0.0-beta");
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/skills/ns/skill/versions/1.0.0-beta",
    );
  });

  it("frontendPublishValidate encodes namespace", async () => {
    await client.frontendPublishValidate("team alpha");
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/skills/team%20alpha/publish/validate",
    );
  });

  it("frontendNamespaceDetail encodes slug", async () => {
    await client.frontendNamespaceDetail("team alpha");
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/namespaces/team%20alpha",
    );
  });

  it("frontendReleaseDetail builds expected path", async () => {
    await client.frontendReleaseDetail("ns", "skill", 42);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/skills/ns/skill/releases/42",
    );
  });

  it("frontendWikiDetail encodes pageSlug", async () => {
    await client.frontendWikiDetail("ns", "skill", "getting started");
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/skills/ns/skill/wiki/getting%20started",
    );
  });

  it("frontendReviews builds expected path", async () => {
    await client.frontendReviews();
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/reviews",
    );
  });

  it("frontendReviews builds query params for page and size", async () => {
    await client.frontendReviews(1, 50);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/reviews?page=1&size=50",
    );
  });

  it("frontendReviewDetail builds expected path", async () => {
    await client.frontendReviewDetail(7);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/reviews/7",
    );
  });

  it("frontendPromotions builds expected path", async () => {
    await client.frontendPromotions();
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/promotions",
    );
  });

  it("frontendPromotions builds query params for page and size", async () => {
    await client.frontendPromotions(0, 50);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/promotions?page=0&size=50",
    );
  });

  it("frontendIssueDetail builds expected path", async () => {
    await client.frontendIssueDetail("ns", "skill", 1);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/skills/ns/skill/issues/1",
    );
  });

  it("frontendDiscussionDetail builds expected path", async () => {
    await client.frontendDiscussionDetail("ns", "skill", 1);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/skills/ns/skill/discussions/1",
    );
  });

  it("frontendProposalDetail builds expected path", async () => {
    await client.frontendProposalDetail("ns", "skill", 1);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/skills/ns/skill/proposals/1",
    );
  });

  it("frontendIssueList accepts optional page and size", async () => {
    await client.frontendIssueList("ns", "skill", 2, 50);
    assert.strictEqual(
      capture.calls.at(-1)!.url,
      "http://localhost:8080/api/v1/frontend/skills/ns/skill/issues?page=2&size=50",
    );
  });
});

// ── SkillHubError tests ──────────────────────────────────────────────────

describe("SkillHubError", () => {
  it("has code, message, and optional status", () => {
    const err = new SkillHubError("not_found", "Skill not found", 404, {
      skillId: 1,
    });
    assert.strictEqual(err.code, "not_found");
    assert.strictEqual(err.message, "Skill not found");
    assert.strictEqual(err.status, 404);
    assert.deepStrictEqual(err.details, { skillId: 1 });
    assert.ok(err instanceof Error);
  });

  it("name is SkillHubError", () => {
    const err = new SkillHubError("test", "msg");
    assert.strictEqual(err.name, "SkillHubError");
  });
});

// ── Unwrap tests ─────────────────────────────────────────────────────────

describe("unwrap", () => {
  let client: SkillHubClient;

  before(() => {
    client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: mockFetch({
        success: true,
        data: { namespaces: [], availableActions: { canCreateNamespace: true } },
      }),
    });
  });

  it("returns data on success envelope", async () => {
    const env: Envelope<{ x: number }> = { success: true, data: { x: 42 } };
    const data = await client.unwrap(env);
    assert.deepStrictEqual(data, { x: 42 });
  });

  it("returns data from resolved promise", async () => {
    const data = await client.unwrap(
      Promise.resolve({ success: true, data: { y: "hello" } } as Envelope<{
        y: string;
      }>),
    );
    assert.deepStrictEqual(data, { y: "hello" });
  });

  it("throws SkillHubError on error envelope", async () => {
    const env: Envelope<unknown> = {
      success: false,
      error: { code: "not_found", message: "skill not found" },
    };
    await assert.rejects(
      () => client.unwrap(env),
      (err: unknown) => {
        assert.ok(err instanceof SkillHubError);
        assert.strictEqual((err as SkillHubError).code, "not_found");
        assert.strictEqual(
          (err as SkillHubError).message,
          "skill not found",
        );
        return true;
      },
    );
  });

  it("throws SkillHubError with client.error when no error code", async () => {
    const env: Envelope<unknown> = {
      success: false,
    };
    await assert.rejects(
      () => client.unwrap(env),
      (err: unknown) => {
        assert.ok(err instanceof SkillHubError);
        assert.strictEqual((err as SkillHubError).code, "client.error");
        return true;
      },
    );
  });

  it("throws SkillHubError on rejected promise (network error)", async () => {
    await assert.rejects(
      () =>
        client.unwrap(
          Promise.reject(new TypeError("fetch failed")),
        ),
      (err: unknown) => {
        assert.ok(err instanceof SkillHubError);
        assert.strictEqual(
          (err as SkillHubError).code,
          "client.network_error",
        );
        return true;
      },
    );
  });

  it("throws SkillHubError on invalid JSON (SyntaxError)", async () => {
    await assert.rejects(
      () =>
        client.unwrap(
          Promise.reject(new SyntaxError("Unexpected token")),
        ),
      (err: unknown) => {
        assert.ok(err instanceof SkillHubError);
        assert.strictEqual(
          (err as SkillHubError).code,
          "client.invalid_json",
        );
        return true;
      },
    );
  });

  it("carries status from HTTP response", async () => {
    const client404 = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: mockFetch(
        { success: false, error: { code: "not_found", message: "gone" } },
        404,
      ),
    });
    try {
      await client404.unwrap(client404.frontendAdmin());
      assert.fail("expected SkillHubError");
    } catch (err) {
      assert.ok(err instanceof SkillHubError);
      assert.strictEqual((err as SkillHubError).status, 404);
    }
  });

  it("preserves SkillHubError through re-wrap", async () => {
    const original = new SkillHubError("custom", "original");
    await assert.rejects(
      () => client.unwrap(Promise.reject(original)),
      (err: unknown) => {
        assert.ok(err instanceof SkillHubError);
        assert.strictEqual((err as SkillHubError).code, "custom");
        return true;
      },
    );
  });
});

// ── Type shape tests (existing, kept for coverage) ────────────────────────

describe("Frontend type shapes", () => {
  it("RegistrySearchReadModel includes availableActions", () => {
    const actions: RegistrySearchActions = {
      canCreateSkill: true,
      canCreateNamespace: true,
      canAccessAdmin: false,
    };
    const model: RegistrySearchReadModel = {
      searchResult: { skillIds: [], total: 0, page: 0, size: 20 },
      featuredLabels: [],
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canCreateSkill, true);
  });

  it("ReviewQueueReadModel includes hasMore", () => {
    const model: ReviewQueueReadModel = {
      tasks: [],
      pendingCount: 0,
      page: 0,
      size: 20,
      hasMore: false,
      availableActions: { canReview: true, canSubmit: true, canWithdraw: true },
    };
    assert.strictEqual(model.hasMore, false);
  });

  it("PromotionQueueReadModel includes hasMore", () => {
    const model: PromotionQueueReadModel = {
      requests: [],
      pendingCount: 0,
      page: 0,
      size: 20,
      hasMore: false,
      availableActions: { canReview: true, canSubmit: true, canWithdraw: true },
    };
    assert.strictEqual(model.hasMore, false);
  });
});

describe("Type exports", () => {
  it("Envelope interface is importable", () => {
    const env: Envelope<Principal> = { success: true, data: undefined };
    assert.strictEqual(env.success, true);
  });

  it("Envelope accepts optional status", () => {
    const env: Envelope<{ x: number }> = {
      success: true,
      data: { x: 1 },
      status: 200,
    };
    assert.strictEqual(env.status, 200);
  });
});

// ── Tool API type shapes ──────────────────────────────────────────────────

describe("Tool API type shapes", () => {
  it("WorkspaceMetadataResponse shape", () => {
    const ws: WorkspaceMetadataResponse = {
      workspace: {
        requiredFiles: ["SKILL.md"],
        optionalFiles: ["README.md"],
        manifestFormat: "SKILL.md with YAML frontmatter",
        schema: { fields: ["name"], required: ["name"] },
      },
    };
    assert.deepStrictEqual(ws.workspace.requiredFiles, ["SKILL.md"]);
  });

  it("PackageManifest shape", () => {
    const m: PackageManifest = {
      entries: [
        {
          path: "SKILL.md",
          size: 100,
          contentType: "text/markdown",
          sha256: "abc123",
        },
      ],
      hash: "sha256:def456",
      totalSize: 100,
      fileCount: 1,
    };
    assert.strictEqual(m.hash, "sha256:def456");
  });

  it("VersionDiff shape", () => {
    const line: DiffLine = { type: "ADD", content: "new line" };
    const hunk: DiffHunk = {
      oldStart: 1,
      oldLines: 0,
      newStart: 1,
      newLines: 1,
      lines: [line],
    };
    const file: DiffFile = {
      path: "a.txt",
      changeType: "MODIFIED",
      oldSize: 10,
      newSize: 12,
      binary: false,
      truncated: false,
      hunks: [hunk],
    };
    const summary: DiffSummary = {
      totalFiles: 1,
      addedFiles: 0,
      modifiedFiles: 1,
      removedFiles: 0,
      addedLines: 1,
      removedLines: 1,
    };
    const diff: VersionDiff = {
      fromVersion: "1.0",
      toVersion: "2.0",
      summary,
      files: [file],
    };
    assert.strictEqual(diff.summary.totalFiles, 1);
  });
});

// ── Pagination iterator tests ────────────────────────────────────────────

describe("Pagination iterators", () => {
  it("iterFrontendReviews stops at maxPages", async () => {
    let callCount = 0;
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: (async () => {
        callCount++;
        return {
          ok: true,
          status: 200,
          json: async () => ({
            success: true,
            data: {
              tasks: [{ id: callCount, skillVersionId: 1, namespaceId: 1, submittedBy: "u1", status: "PENDING", submittedAt: "2025-01-01T00:00:00Z" }],
              pendingCount: 10,
              page: callCount - 1,
              size: 20,
              hasMore: true,
              availableActions: { canReview: true, canSubmit: true, canWithdraw: true },
            },
          }),
          headers: new Headers(),
        } as Response;
      }) as typeof globalThis.fetch,
    });

    const pages: ReviewQueueReadModel[] = [];
    for await (const page of client.iterFrontendReviews({
      maxPages: 3,
      size: 1,
    })) {
      pages.push(page);
    }
    assert.strictEqual(pages.length, 3);
    assert.strictEqual(callCount, 3);
  });

  it("iterFrontendReviews stops when hasMore is false", async () => {
    let callCount = 0;
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: (async () => {
        callCount++;
        return {
          ok: true,
          status: 200,
          json: async () => ({
            success: true,
            data: {
              tasks: [],
              pendingCount: 0,
              page: 0,
              size: 20,
              hasMore: false,
              availableActions: { canReview: true, canSubmit: true, canWithdraw: true },
            },
          }),
          headers: new Headers(),
        } as Response;
      }) as typeof globalThis.fetch,
    });

    const pages: ReviewQueueReadModel[] = [];
    for await (const page of client.iterFrontendReviews()) {
      pages.push(page);
    }
    assert.strictEqual(pages.length, 1);
    assert.strictEqual(callCount, 1);
  });

  it("iterFrontendReviews stops when tasks array is empty", async () => {
    let callCount = 0;
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: (async () => {
        callCount++;
        return {
          ok: true,
          status: 200,
          json: async () => ({
            success: true,
            data: {
              tasks: [],
              pendingCount: 0,
              page: callCount - 1,
              size: 20,
              hasMore: true,
              availableActions: { canReview: true, canSubmit: true, canWithdraw: true },
            },
          }),
          headers: new Headers(),
        } as Response;
      }) as typeof globalThis.fetch,
    });

    const pages: ReviewQueueReadModel[] = [];
    for await (const page of client.iterFrontendReviews()) {
      pages.push(page);
    }
    assert.strictEqual(pages.length, 1);
    assert.strictEqual(callCount, 1);
  });

  it("iterFrontendSearch increments page", async () => {
    const pagesRequested: number[] = [];
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: (async (input: RequestInfo | URL) => {
        const url = typeof input === "string" ? input : input.toString();
        const pageMatch = url.match(/page=(\d+)/);
        const page = pageMatch ? parseInt(pageMatch[1], 10) : 0;
        pagesRequested.push(page);
        return {
          ok: true,
          status: 200,
          json: async () => ({
            success: true,
            data: {
              searchResult: { skillIds: [1], total: 3, page, size: 1 },
              featuredLabels: [],
              availableActions: { canCreateSkill: true, canCreateNamespace: true, canAccessAdmin: false },
            },
          }),
          headers: new Headers(),
        } as Response;
      }) as typeof globalThis.fetch,
    });

    const pages: RegistrySearchReadModel[] = [];
    for await (const page of client.iterFrontendSearch({
      size: 1,
      maxPages: 3,
    })) {
      pages.push(page);
    }
    assert.strictEqual(pages.length, 3);
    assert.deepStrictEqual(pagesRequested, [0, 1, 2]);
  });

  it("iterFrontendSearch preserves original query filters", async () => {
    let capturedUrl = "";
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: (async (input: RequestInfo | URL) => {
        capturedUrl = typeof input === "string" ? input : input.toString();
        return {
          ok: true,
          status: 200,
          json: async () => ({
            success: true,
            data: {
              searchResult: { skillIds: [], total: 0, page: 0, size: 20 },
              featuredLabels: [],
              availableActions: { canCreateSkill: true, canCreateNamespace: true, canAccessAdmin: false },
            },
          }),
          headers: new Headers(),
        } as Response;
      }) as typeof globalThis.fetch,
    });

    for await (const _page of client.iterFrontendSearch({
      keyword: "agent",
      sortBy: "downloads",
      installableOnly: true,
      maxPages: 1,
    })) {
      // drain
    }
    assert.ok(capturedUrl.includes("q=agent"));
    assert.ok(capturedUrl.includes("sort=downloads"));
    assert.ok(capturedUrl.includes("installable=true"));
  });

  it("iterFrontendPromotions stops at maxPages", async () => {
    let callCount = 0;
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: (async () => {
        callCount++;
        return {
          ok: true,
          status: 200,
          json: async () => ({
            success: true,
            data: {
              requests: [{ id: callCount, sourceSkillId: 1, sourceVersionId: 1, targetNamespaceId: 1, submittedBy: "u1", status: "PENDING", submittedAt: "2025-01-01T00:00:00Z" }],
              pendingCount: 10,
              page: callCount - 1,
              size: 20,
              hasMore: true,
              availableActions: { canReview: true, canSubmit: true, canWithdraw: true },
            },
          }),
          headers: new Headers(),
        } as Response;
      }) as typeof globalThis.fetch,
    });

    const pages: PromotionQueueReadModel[] = [];
    for await (const page of client.iterFrontendPromotions({ maxPages: 2 })) {
      pages.push(page);
    }
    assert.strictEqual(pages.length, 2);
    assert.strictEqual(callCount, 2);
  });

  it("iterFrontendIssues stops when totalCount exhausted", async () => {
    let callCount = 0;
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: (async () => {
        callCount++;
        return {
          ok: true,
          status: 200,
          json: async () => ({
            success: true,
            data: {
              issues: [{ id: callCount, title: "Test", status: "OPEN", authorId: "u1", locked: false, commentCount: 0 }],
              totalCount: 2,
              page: 0,
              size: 2,
              availableActions: { canCreateIssue: true },
            },
          }),
          headers: new Headers(),
        } as Response;
      }) as typeof globalThis.fetch,
    });

    const pages: IssueListReadModel[] = [];
    for await (const page of client.iterFrontendIssues("ns", "skill", {
      size: 2,
    })) {
      pages.push(page);
    }
    assert.strictEqual(pages.length, 1);
  });

  it("iterReleases stops at maxPages", async () => {
    let callCount = 0;
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: (async () => {
        callCount++;
        return {
          ok: true,
          status: 200,
          json: async () => ({
            success: true,
            data: {
              releases: [{ id: callCount, versionId: 1, channel: "stable", title: "v1", draft: false, prerelease: false, yanked: false, publisherId: "u1" }],
              totalCount: 10,
              page: callCount - 1,
              size: 1,
            },
          }),
          headers: new Headers(),
        } as Response;
      }) as typeof globalThis.fetch,
    });

    const pages: unknown[] = [];
    for await (const page of client.iterReleases("ns", "skill", {
      maxPages: 2,
      size: 1,
    })) {
      pages.push(page);
    }
    assert.strictEqual(pages.length, 2);
  });

  it("iterPipelineRuns increments page", async () => {
    const pagesRequested: number[] = [];
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: (async (input: RequestInfo | URL) => {
        const url = typeof input === "string" ? input : input.toString();
        const pageMatch = url.match(/page=(\d+)/);
        const page = pageMatch ? parseInt(pageMatch[1], 10) : 0;
        pagesRequested.push(page);
        return {
          ok: true,
          status: 200,
          json: async () => ({
            success: true,
            data: {
              runs: [{ id: page + 1, pipelineId: 1, skillId: 1, triggerType: "publish", triggeredBy: "u1", status: "COMPLETED", checkCount: 3, passedCount: 3, failedCount: 0, skippedCount: 0, createdAt: "2025-01-01T00:00:00Z", updatedAt: "2025-01-01T00:00:00Z" }],
              totalCount: 3,
              page,
              size: 1,
            },
          }),
          headers: new Headers(),
        } as Response;
      }) as typeof globalThis.fetch,
    });

    const pages: PipelineRunListResult[] = [];
    for await (const page of client.iterPipelineRuns(1, {
      size: 1,
      maxPages: 3,
    })) {
      pages.push(page);
    }
    assert.strictEqual(pages.length, 3);
    assert.deepStrictEqual(pagesRequested, [0, 1, 2]);
  });

  it("iterFrontendDiscussions stops on empty list", async () => {
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: mockFetch({
        success: true,
        data: {
          discussions: [],
          totalCount: 0,
          page: 0,
          size: 20,
          availableActions: { canCreateDiscussion: true },
        },
      }),
    });

    const pages: DiscussionListReadModel[] = [];
    for await (const page of client.iterFrontendDiscussions("ns", "skill")) {
      pages.push(page);
    }
    assert.strictEqual(pages.length, 1);
  });

  it("iterFrontendProposals stops on empty list", async () => {
    const client = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: mockFetch({
        success: true,
        data: {
          proposals: [],
          totalCount: 0,
          page: 0,
          size: 20,
          availableActions: { canCreateProposal: true },
        },
      }),
    });

    const pages: ProposalListReadModel[] = [];
    for await (const page of client.iterFrontendProposals("ns", "skill")) {
      pages.push(page);
    }
    assert.strictEqual(pages.length, 1);
  });
});

// ── Envelope handling (HTTP-level tests) ─────────────────────────────────

describe("Envelope handling", () => {
  let originalFetch: typeof globalThis.fetch;
  const client = new SkillHubClient({
    baseUrl: "http://localhost:8080",
    fetch: mockFetch({
      success: true,
      data: {},
    }),
  });

  before(() => {
    originalFetch = globalThis.fetch;
  });

  after(() => {
    globalThis.fetch = originalFetch;
  });

  it("returns error envelope when fetch resolves to error response", async () => {
    const errorClient = new SkillHubClient({
      baseUrl: "http://localhost:8080",
      fetch: (async () => {
        return {
          ok: true,
          status: 200,
          json: async () => ({
            success: false,
            error: {
              code: "search.failed",
              message: "search is unavailable",
            },
          }),
        } as Response;
      }) as typeof globalThis.fetch,
    });

    const result = await errorClient.frontendSearch({ keyword: "x" });
    assert.strictEqual(result.success, false);
    assert.strictEqual(result.error?.code, "search.failed");
  });
});
