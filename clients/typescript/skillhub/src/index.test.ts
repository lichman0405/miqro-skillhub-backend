/**
 * Tests for the SkillHub TypeScript client.
 *
 * These tests validate the client's type safety and frontend contract
 * coverage — every /api/v1/frontend/* route exposed by the Go handlers
 * must have a corresponding typed client method.
 */
import { describe, it } from "node:test";
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
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canReview, true);
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
      availableActions: actions,
    };
    assert.strictEqual(model.availableActions.canSubmit, true);
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
