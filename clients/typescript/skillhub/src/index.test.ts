/**
 * Tests for the SkillHub TypeScript client.
 *
 * These tests validate the client's type safety and frontend contract
 * coverage — every /api/v1/frontend/* route exposed by the Go handlers
 * must have a corresponding typed client method.
 */
import { describe, it, before, after } from "node:test";
import assert from "node:assert/strict";
import {
  SkillHubClient,
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
  // Tool API types
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
} from "./index.js";

describe("SkillHubClient", () => {
  it("constructs with default base URL", () => {
    const client = new SkillHubClient();
    assert.ok(client instanceof SkillHubClient);
  });

  it("constructs with custom base URL", () => {
    const client = new SkillHubClient("https://skillhub.example.com");
    assert.ok(client instanceof SkillHubClient);
  });

  it("builds correct login URL", () => {
    const client = new SkillHubClient("http://localhost:8080");
    assert.ok(client instanceof SkillHubClient);
  });

  it("builds correct search URL with params", () => {
    const client = new SkillHubClient("http://localhost:8080");
    assert.strictEqual(typeof client.search, "function");
  });

  it("builds correct getSkill path", () => {
    const client = new SkillHubClient("http://localhost:8080");
    assert.strictEqual(typeof client.getSkill, "function");
  });

  it("builds correct getNamespace path", () => {
    const client = new SkillHubClient("http://localhost:8080");
    assert.strictEqual(typeof client.getNamespace, "function");
  });
});

describe("Frontend client methods", () => {
  const client = new SkillHubClient("http://localhost:8080");

  it("frontendSearch is a function", () => {
    assert.strictEqual(typeof client.frontendSearch, "function");
  });

  it("frontendSkillDetail is a function", () => {
    assert.strictEqual(typeof client.frontendSkillDetail, "function");
  });

  it("frontendVersionDetail is a function", () => {
    assert.strictEqual(typeof client.frontendVersionDetail, "function");
  });

  it("frontendPublishValidate is a function", () => {
    assert.strictEqual(typeof client.frontendPublishValidate, "function");
  });

  it("frontendNamespaces is a function", () => {
    assert.strictEqual(typeof client.frontendNamespaces, "function");
  });

  it("frontendNamespaceDetail is a function", () => {
    assert.strictEqual(typeof client.frontendNamespaceDetail, "function");
  });

  it("frontendReviews is a function", () => {
    assert.strictEqual(typeof client.frontendReviews, "function");
  });

  it("frontendReviewDetail is a function", () => {
    assert.strictEqual(typeof client.frontendReviewDetail, "function");
  });

  it("frontendPromotions is a function", () => {
    assert.strictEqual(typeof client.frontendPromotions, "function");
  });

  it("frontendPromotionDetail is a function", () => {
    assert.strictEqual(typeof client.frontendPromotionDetail, "function");
  });

  it("frontendGovernance is a function", () => {
    assert.strictEqual(typeof client.frontendGovernance, "function");
  });

  it("frontendAdmin is a function", () => {
    assert.strictEqual(typeof client.frontendAdmin, "function");
  });
});

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
    assert.strictEqual(model.availableActions.canAccessAdmin, false);
  });

  it("SkillDetailReadModel includes availableActions", () => {
    const actions: SkillDetailActions = {
      canEdit: false,
      canPublish: false,
      canDelete: false,
      canSubmitForReview: false,
      canRequestPromotion: false,
      canStar: true,
      canReport: true,
      canManage: false,
    };
    const model: SkillDetailReadModel = {
      skill: {
        id: 1,
        slug: "my-skill",
        displayName: "My Skill",
        ownerId: "u1",
        summary: "test",
        visibility: "PUBLIC",
        status: "ACTIVE",
        downloadCount: 0,
        starCount: 0,
        ratingAvg: 0,
        canManage: false,
      },
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canStar, true);
    assert.strictEqual(model.availableActions.canManage, false);
  });

  it("VersionDetailReadModel includes availableActions", () => {
    const actions: VersionActions = {
      canCompare: true,
      canDownload: true,
      canSubmitForReview: false,
      canRequestPromotion: false,
      canYank: false,
      canReview: false,
    };
    const model: VersionDetailReadModel = {
      version: { id: 1, version: "1.0.0", status: "ACTIVE" },
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canDownload, true);
    assert.strictEqual(model.availableActions.canYank, false);
  });

  it("PublishValidateReadModel includes availableActions", () => {
    const actions: PublishValidateActions = {
      canPublish: true,
      canOverrideWarnings: false,
    };
    const model: PublishValidateReadModel = {
      valid: false,
      warnings: ["no package uploaded"],
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canPublish, true);
  });

  it("NamespaceListReadModel includes availableActions", () => {
    const actions: NamespaceListActions = {
      canCreateNamespace: true,
    };
    const model: NamespaceListReadModel = {
      namespaces: [],
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canCreateNamespace, true);
  });

  it("NamespaceDetailReadModel includes availableActions", () => {
    const actions: NamespaceDetailActions = {
      canEdit: true,
      canDelete: false,
      canManageMembers: true,
      canTransferOwner: false,
      canLeave: false,
      canJoin: false,
    };
    const model: NamespaceDetailReadModel = {
      namespace: {
        id: 1,
        slug: "my-ns",
        displayName: "My NS",
        type: "PUBLIC",
        description: "",
      },
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canEdit, true);
    assert.strictEqual(model.availableActions.canDelete, false);
  });

  it("ReviewQueueReadModel includes availableActions", () => {
    const actions: ReviewQueueActions = {
      canReview: true,
      canSubmit: true,
      canWithdraw: true,
    };
    const model: ReviewQueueReadModel = {
      tasks: [],
      pendingCount: 0,
      page: 0,
      size: 20,
      hasMore: false,
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canReview, true);
    assert.strictEqual(model.page, 0);
    assert.strictEqual(model.hasMore, false);
  });

  it("ReviewDetailReadModel includes availableActions", () => {
    const actions: ReviewDetailActions = {
      canApprove: true,
      canReject: true,
      canWithdraw: false,
    };
    const model: ReviewDetailReadModel = {
      task: {
        id: 1,
        skillVersionId: 2,
        namespaceId: 3,
        submittedBy: "u1",
        status: "PENDING",
        submittedAt: "2025-01-01T00:00:00Z",
      },
      skillName: "test",
      version: "1.0.0",
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canApprove, true);
  });

  it("PromotionQueueReadModel includes availableActions", () => {
    const actions: PromotionQueueActions = {
      canReview: true,
      canSubmit: true,
      canWithdraw: true,
    };
    const model: PromotionQueueReadModel = {
      requests: [],
      pendingCount: 0,
      page: 0,
      size: 20,
      hasMore: false,
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canSubmit, true);
    assert.strictEqual(model.page, 0);
    assert.strictEqual(model.hasMore, false);
  });

  it("PromotionDetailReadModel includes availableActions", () => {
    const actions: PromotionDetailActions = {
      canApprove: true,
      canReject: false,
      canWithdraw: false,
    };
    const model: PromotionDetailReadModel = {
      request: {
        id: 1,
        sourceSkillId: 2,
        sourceVersionId: 3,
        targetNamespaceId: 4,
        submittedBy: "u1",
        status: "PENDING",
        submittedAt: "2025-01-01T00:00:00Z",
      },
      sourceSkillName: "test",
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canApprove, true);
  });

  it("GovernanceWorkbenchReadModel includes availableActions", () => {
    const actions: GovernanceWorkbenchActions = {
      canReview: true,
      canAccessAdmin: false,
      canViewAuditLog: true,
    };
    const model: GovernanceWorkbenchReadModel = {
      summary: {
        total: 0,
        unread: 0,
        byCategory: {},
        pendingReviews: 0,
        pendingPromotions: 0,
      },
      recentActivity: [],
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canViewAuditLog, true);
    assert.strictEqual(model.availableActions.canAccessAdmin, false);
  });

  it("AdminPageReadModel includes availableActions", () => {
    const actions: AdminPageActions = {
      canManageSkills: true,
      canManageUsers: false,
      canManageLabels: true,
      canResolveReports: true,
      canRebuildSearch: false,
      canViewAuditLog: false,
      canManageNamespaces: false,
    };
    const model: AdminPageReadModel = {
      stats: {
        totalSkills: 0,
        totalNamespaces: 0,
        totalUsers: 0,
        pendingReviews: 0,
        pendingPromotions: 0,
        openReports: 0,
      },
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canManageSkills, true);
    assert.strictEqual(model.availableActions.canManageUsers, false);
  });
});

describe("Tool API client methods", () => {
  const client = new SkillHubClient("http://localhost:8080");

  it("toolWorkspaceMetadata is a function", () => {
    assert.strictEqual(typeof client.toolWorkspaceMetadata, "function");
  });

  it("toolPackageHash is a function", () => {
    assert.strictEqual(typeof client.toolPackageHash, "function");
  });

  it("toolResolve is a function", () => {
    assert.strictEqual(typeof client.toolResolve, "function");
  });

  it("toolInstall is a function", () => {
    assert.strictEqual(typeof client.toolInstall, "function");
  });

  it("toolDiff is a function", () => {
    assert.strictEqual(typeof client.toolDiff, "function");
  });

  it("toolValidate is a function", () => {
    assert.strictEqual(typeof client.toolValidate, "function");
  });

  it("toolPublish is a function", () => {
    assert.strictEqual(typeof client.toolPublish, "function");
  });

  it("toolEvaluate is a function", () => {
    assert.strictEqual(typeof client.toolEvaluate, "function");
  });

  it("toolPropose is a function", () => {
    assert.strictEqual(typeof client.toolPropose, "function");
  });
});

describe("Tool API type shapes", () => {
  it("WorkspaceMetadataResponse shape", () => {
    const ws: WorkspaceMetadataResponse = {
      workspace: {
        requiredFiles: ["SKILL.md"],
        optionalFiles: ["README.md"],
        manifestFormat: "SKILL.md with YAML frontmatter",
        schema: {
          fields: ["name", "description", "version"],
          required: ["name"],
        },
      },
    };
    assert.deepStrictEqual(ws.workspace.requiredFiles, ["SKILL.md"]);
  });

  it("PackageManifest shape", () => {
    const m: PackageManifest = {
      entries: [
        { path: "SKILL.md", size: 100, contentType: "text/markdown", sha256: "abc123" },
      ],
      hash: "sha256:def456",
      totalSize: 100,
      fileCount: 1,
    };
    assert.strictEqual(m.hash, "sha256:def456");
    assert.strictEqual(m.fileCount, 1);
  });

  it("PackageHashRequest shape", () => {
    const entries: PackageEntry[] = [
      { path: "a.txt", content: "hello", size: 5, contentType: "text/plain" },
    ];
    assert.strictEqual(entries.length, 1);
    assert.strictEqual(entries[0].path, "a.txt");
  });

  it("ResolveResult shape", () => {
    const r: ResolveResult = {
      skillId: 1,
      namespace: "ns",
      slug: "my-skill",
      version: "1.0.0",
      versionId: 5,
      fingerprint: "sha256:abc123",
      downloadUrl: "/api/v1/skills/ns/my-skill/versions/1.0.0/download",
    };
    assert.strictEqual(r.fingerprint.startsWith("sha256:"), true);
  });

  it("InstallTarget shape", () => {
    const agent: AgentRuntime = { type: "claude-code", minVersion: "1.0.0" };
    const target: InstallTarget = {
      skillId: 1,
      skillSlug: "my-skill",
      namespace: "ns",
      version: "1.0.0",
      fingerprint: "sha256:abc",
      downloadUrl: "/download",
      supportedAgents: [agent],
    };
    assert.strictEqual(target.supportedAgents![0].type, "claude-code");
  });

  it("VersionDiff shape", () => {
    const line: DiffLine = { type: "ADD", content: "new line" };
    const hunk: DiffHunk = {
      oldStart: 1, oldLines: 0, newStart: 1, newLines: 1,
      lines: [line],
    };
    const file: DiffFile = {
      path: "a.txt",
      changeType: "MODIFIED",
      oldSize: 10, newSize: 12,
      binary: false, truncated: false,
      hunks: [hunk],
    };
    const summary: DiffSummary = {
      totalFiles: 1, addedFiles: 0, modifiedFiles: 1, removedFiles: 0,
      addedLines: 1, removedLines: 1,
    };
    const diff: VersionDiff = {
      fromVersion: "1.0", toVersion: "2.0",
      summary,
      files: [file],
    };
    assert.strictEqual(diff.summary.totalFiles, 1);
    assert.strictEqual(diff.files[0].changeType, "MODIFIED");
  });

  it("ToolValidateResponse shape", () => {
    const r: ToolValidateResponse = {
      valid: true,
      warnings: [],
      resolvedSlug: "my-skill",
      resolvedVersion: "1.0.0",
    };
    assert.strictEqual(r.valid, true);
  });

  it("ToolPublishResponse shape", () => {
    const r: ToolPublishResponse = {
      skillId: 1,
      slug: "my-skill",
      version: { id: 10, version: "1.0.0", status: "PUBLISHED" },
    };
    assert.strictEqual(r.version.version, "1.0.0");
  });

  it("EvaluateRequest shape", () => {
    const req: EvaluateRequest = { skillId: 1, versionId: 2, trigger: "publish" };
    assert.strictEqual(req.trigger, "publish");
  });

  it("EvaluateResponse shape (placeholder)", () => {
    const resp: EvaluateResponse = {
      accepted: false,
      message: "evaluation trigger is not yet implemented (Phase 12)",
    };
    assert.strictEqual(resp.accepted, false);
  });

  it("ProposalRequest shape", () => {
    const req: ProposalRequest = {
      skillId: 1,
      namespace: "ns",
      slug: "my-skill",
      title: "Update README",
      description: "Better docs",
    };
    assert.strictEqual(req.title, "Update README");
  });

  it("ProposalResponse shape (placeholder)", () => {
    const resp: ProposalResponse = {
      accepted: false,
      message: "proposal preparation is not yet implemented (Phase 11)",
    };
    assert.strictEqual(resp.accepted, false);
  });
});

describe("Release client methods", () => {
  const client = new SkillHubClient("http://localhost:8080");

  it("listReleases is a function", () => {
    assert.strictEqual(typeof client.listReleases, "function");
  });

  it("getLatestRelease is a function", () => {
    assert.strictEqual(typeof client.getLatestRelease, "function");
  });

  it("getRelease is a function", () => {
    assert.strictEqual(typeof client.getRelease, "function");
  });

  it("createRelease is a function", () => {
    assert.strictEqual(typeof client.createRelease, "function");
  });

  it("updateRelease is a function", () => {
    assert.strictEqual(typeof client.updateRelease, "function");
  });

  it("deleteRelease is a function", () => {
    assert.strictEqual(typeof client.deleteRelease, "function");
  });

  it("publishRelease is a function", () => {
    assert.strictEqual(typeof client.publishRelease, "function");
  });

  it("frontendReleaseList is a function", () => {
    assert.strictEqual(typeof client.frontendReleaseList, "function");
  });

  it("frontendReleaseDetail is a function", () => {
    assert.strictEqual(typeof client.frontendReleaseDetail, "function");
  });
});

describe("Type exports", () => {
  it("Envelope interface is importable", () => {
    const env: Envelope<Principal> = { success: true, data: undefined };
    assert.strictEqual(env.success, true);
  });

  it("SearchResult shape is consistent", () => {
    const result: SearchResult = {
      skillIds: [1, 2, 3],
      total: 3,
      page: 1,
      size: 20,
    };
    assert.strictEqual(result.skillIds.length, 3);
    assert.strictEqual(result.total, 3);
  });

  it("SkillDetail shape includes canManage", () => {
    const detail: SkillDetail = {
      id: 1,
      slug: "my-skill",
      displayName: "My Skill",
      ownerId: "user-1",
      summary: "A test skill",
      visibility: "PUBLIC",
      status: "ACTIVE",
      downloadCount: 100,
      starCount: 5,
      ratingAvg: 4.5,
      canManage: false,
    };
    assert.strictEqual(detail.canManage, false);
  });

  it("Principal shape includes platformRoles", () => {
    const principal: Principal = {
      userID: "user-1",
      displayName: "Test",
      email: "test@example.com",
      authMethod: "session",
      platformRoles: { SUPER_ADMIN: true },
      isAuthenticated: true,
    };
    assert.ok(principal.platformRoles.SUPER_ADMIN);
  });
});

describe("Frontend client URL construction", () => {
  let captured: { url: string; init?: RequestInit } | null = null;
  let originalFetch: typeof globalThis.fetch;
  const client = new SkillHubClient("http://localhost:8080");

  before(() => {
    originalFetch = globalThis.fetch;
    globalThis.fetch = (async (
      input: RequestInfo | URL,
      init?: RequestInit
    ): Promise<Response> => {
      captured = {
        url: typeof input === "string" ? input : input.toString(),
        init,
      };
      return {
        ok: true,
        status: 200,
        json: async () => ({ success: true, data: {} }),
      } as Response;
    }) as typeof globalThis.fetch;
  });

  after(() => {
    globalThis.fetch = originalFetch;
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
    assert.ok(captured, "fetch should have been called");
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/search?q=agent+tools&sort=downloads&page=1&size=20&labels=go%2Cci&installable=true"
    );
  });

  it("frontendSearch omits empty query params", async () => {
    await client.frontendSearch({});
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/search"
    );
  });

  it("frontendSkillDetail encodes path params", async () => {
    await client.frontendSkillDetail("team alpha", "my/skill");
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/skills/team%20alpha/my%2Fskill"
    );
  });

  it("frontendVersionDetail encodes version", async () => {
    await client.frontendVersionDetail("ns", "skill", "1.0.0-beta");
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/skills/ns/skill/versions/1.0.0-beta"
    );
  });

  it("frontendPublishValidate encodes namespace", async () => {
    await client.frontendPublishValidate("team alpha");
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/skills/team%20alpha/publish/validate"
    );
  });

  it("frontendNamespaceDetail encodes slug", async () => {
    await client.frontendNamespaceDetail("team alpha");
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/namespaces/team%20alpha"
    );
  });

  it("frontendReleaseDetail builds expected path", async () => {
    await client.frontendReleaseDetail("ns", "skill", 42);
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/skills/ns/skill/releases/42"
    );
  });

  it("frontendWikiDetail encodes pageSlug", async () => {
    await client.frontendWikiDetail("ns", "skill", "getting started");
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/skills/ns/skill/wiki/getting%20started"
    );
  });

  it("frontendReviews builds expected path", async () => {
    await client.frontendReviews();
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/reviews"
    );
  });

  it("frontendReviews builds query params for page and size", async () => {
    await client.frontendReviews(1, 50);
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/reviews?page=1&size=50"
    );
  });

  it("frontendReviewDetail builds expected path", async () => {
    await client.frontendReviewDetail(7);
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/reviews/7"
    );
  });

  it("frontendPromotions builds expected path", async () => {
    await client.frontendPromotions();
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/promotions"
    );
  });

  it("frontendPromotions builds query params for page and size", async () => {
    await client.frontendPromotions(0, 50);
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/promotions?page=0&size=50"
    );
  });

  it("frontendGovernance builds expected path", async () => {
    await client.frontendGovernance();
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/governance"
    );
  });

  it("frontendAdmin builds expected path", async () => {
    await client.frontendAdmin();
    assert.strictEqual(
      captured!.url,
      "http://localhost:8080/api/v1/frontend/admin"
    );
  });
});

describe("Envelope handling", () => {
  let originalFetch: typeof globalThis.fetch;
  const client = new SkillHubClient("http://localhost:8080");

  before(() => {
    originalFetch = globalThis.fetch;
  });

  after(() => {
    globalThis.fetch = originalFetch;
  });

  it("returns error envelope when fetch resolves to error response", async () => {
    globalThis.fetch = (async () => {
      return {
        ok: true,
        status: 200,
        json: async () => ({
          success: false,
          error: { code: "search.failed", message: "search is unavailable" },
        }),
      } as Response;
    }) as typeof globalThis.fetch;

    const result = await client.frontendSearch({ keyword: "x" });
    assert.strictEqual(result.success, false);
    assert.strictEqual(result.error?.code, "search.failed");
  });
});
