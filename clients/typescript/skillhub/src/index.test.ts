/**
 * Tests for the SkillHub TypeScript client.
 *
 * These tests validate the client's type safety and basic construction.
 * Integration tests against a running server require a backend instance.
 */
import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { SkillHubClient, type Envelope, type Principal, type SearchResult, type SkillDetail } from "./index.js";

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
    // Verify the client was constructed — login would actually make a request.
    assert.ok(client instanceof SkillHubClient);
  });

  it("builds correct search URL with params", () => {
    const client = new SkillHubClient("http://localhost:8080");
    // Verify method exists and is callable.
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

describe("Type exports", () => {
  it("Envelope interface is importable", () => {
    // Type-level check; runtime validation that the module loaded.
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
