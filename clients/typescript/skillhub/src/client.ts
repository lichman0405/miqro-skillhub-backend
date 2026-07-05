/**
 * SkillHubClient — typed HTTP wrapper over the SkillHub backend.
 *
 * Implements SkillHubTransport for domain function delegation.
 * All public endpoint methods delegate to domain modules under src/domains/.
 */

import {
  type SkillHubTransport,
  type SkillHubClientConfig,
  type Envelope,
  SkillHubError,
  encodePath,
  encodeQuery,
} from "./core.js";
import type { SearchQuery, SearchResult, SkillDetail } from "./types/common.js";
import type { Principal } from "./types/common.js";
import type {
  RegistrySearchReadModel,
  SkillDetailReadModel,
  VersionDetailReadModel,
  PublishValidateReadModel,
  NamespaceListReadModel,
  NamespaceDetailReadModel,
  ReviewQueueReadModel,
  ReviewDetailReadModel,
  PromotionQueueReadModel,
  PromotionDetailReadModel,
  GovernanceWorkbenchReadModel,
  AdminPageReadModel,
} from "./types/frontend.js";
import type {
  WorkspaceMetadataResponse,
  PackageEntry,
  PackageHashResponse,
  ResolveResult,
  InstallTarget,
  VersionDiff,
  ToolValidateResponse,
  ToolPublishResponse,
  EvaluateRequest,
  EvaluateResponse,
  ProposalRequest,
  ProposalResponse,
} from "./types/tooling.js";
import type {
  Release,
  ReleaseListResult,
  ReleaseDetailResponse,
  CreateReleaseRequest,
  UpdateReleaseRequest,
  ReleaseListReadModel,
  ReleaseDetailReadModel,
} from "./types/releases.js";
import type {
  CreateIssueRequest,
  UpdateIssueRequest,
  CommunitySearchResult,
  IssueListReadModel,
  IssueDetailReadModel,
  DiscussionListReadModel,
  DiscussionDetailReadModel,
  WikiPageListReadModel,
  WikiPageDetailReadModel,
  ProposalListReadModel,
  ProposalDetailReadModel,
} from "./types/community.js";
import type {
  PipelineRun,
  PipelineRunListResult,
  CheckRun,
  CheckArtifact,
  GateEvalResult,
} from "./types/agentci.js";
import type {
  ReviewMutationRequest,
  ReviewMutationResponse,
  PromotionMutationRequest,
  PromotionMutationResponse,
  WithdrawResponse,
} from "./types/mutations.js";
import type { PageIteratorOptions } from "./pagination.js";

import * as authApi from "./domains/auth.js";
import * as portalApi from "./domains/portal.js";
import * as frontendApi from "./domains/frontend.js";
import * as toolingApi from "./domains/tooling.js";
import * as releasesApi from "./domains/releases.js";
import * as communityApi from "./domains/community.js";
import * as agentciApi from "./domains/agentci.js";
import * as reviewApi from "./domains/review.js";
import * as promotionApi from "./domains/promotion.js";

/** SkillHub API client — thin HTTP wrapper over the backend. */
export class SkillHubClient implements SkillHubTransport {
  private baseUrl: string;
  private customFetch?: typeof fetch;
  private defaultCredentials?: RequestCredentials;
  private token?: string;
  private getToken?: () => string | undefined | Promise<string | undefined>;
  private customHeaders?: HeadersInit;

  constructor(config: SkillHubClientConfig = {}) {
    if (typeof config === "string") {
      this.baseUrl = config.replace(/\/+$/, "");
    } else {
      this.baseUrl = (config.baseUrl ?? "http://localhost:8080").replace(
        /\/+$/,
        "",
      );
      this.customFetch = config.fetch;
      this.defaultCredentials = config.credentials;
      this.token = config.token;
      this.getToken = config.getToken;
      this.customHeaders = config.headers;
    }
  }

  // ── Transport implementation ──────────────────────────────────────────

  /** Build a URL-safe path from segments, encoding each one. */
  path(...parts: Array<string | number>): string {
    return encodePath(...parts);
  }

  /** Build a query string from a params record. */
  query(
    params: Record<string, string | number | boolean | string[] | undefined>,
  ): string {
    return encodeQuery(params);
  }

  /** Execute an HTTP request and return the JSON envelope. */
  async request<T>(path: string, init?: RequestInit): Promise<Envelope<T>> {
    const headers = new Headers(this.customHeaders);

    const isFormData = init?.body instanceof FormData;
    const reqHeaders = init?.headers ? new Headers(init.headers) : null;
    const hasContentType =
      headers.has("content-type") || reqHeaders?.has("content-type");
    if (!hasContentType && !isFormData) {
      headers.set("Content-Type", "application/json");
    }

    if (reqHeaders) {
      reqHeaders.forEach((v, k) => headers.set(k, v));
    }

    if (this.token) {
      headers.set("Authorization", `Bearer ${this.token}`);
    } else if (this.getToken) {
      const t = await this.getToken();
      if (t) headers.set("Authorization", `Bearer ${t}`);
    }

    const fetchInit: RequestInit = { ...init, headers };

    if (init?.credentials) {
      fetchInit.credentials = init.credentials;
    } else if (this.defaultCredentials !== undefined) {
      fetchInit.credentials = this.defaultCredentials;
    }

    const fetchFn = this.customFetch ?? globalThis.fetch;
    const res = await fetchFn(`${this.baseUrl}${path}`, fetchInit);

    let body: Record<string, unknown>;
    try {
      body = (await res.json()) as Record<string, unknown>;
    } catch {
      body = {
        success: false,
        error: {
          code: "client.invalid_json",
          message: `Invalid JSON response (status ${res.status})`,
        },
      };
    }

    if (body && typeof body === "object") {
      (body as unknown as Record<string, unknown>).status = res.status;
    }

    return body as unknown as Envelope<T>;
  }

  // ── Unwrap helper ─────────────────────────────────────────────────────

  async unwrap<T>(
    input: Promise<Envelope<T>> | Envelope<T>,
  ): Promise<T> {
    let env: Envelope<T>;
    try {
      env = await input;
    } catch (err: unknown) {
      if (err instanceof SkillHubError) throw err;
      const message = err instanceof Error ? err.message : String(err);
      if (err instanceof TypeError) {
        throw new SkillHubError("client.network_error", message);
      }
      throw new SkillHubError("client.invalid_json", message);
    }

    if (!env.success || env.error) {
      throw new SkillHubError(
        env.error?.code ?? "client.error",
        env.error?.message ?? "Unknown error",
        (env as unknown as Record<string, unknown>).status as number | undefined,
        env.error?.details,
      );
    }

    return env.data as T;
  }

  // ── Auth ──────────────────────────────────────────────────────────────

  async login(username: string, password: string) {
    return authApi.login(this, username, password);
  }

  async me(): Promise<Envelope<Principal>> {
    return authApi.me(this);
  }

  // ── Search ────────────────────────────────────────────────────────────

  async search(query: SearchQuery): Promise<Envelope<SearchResult>> {
    return portalApi.search(this, query);
  }

  // ── Portal detail ─────────────────────────────────────────────────────

  async getSkill(namespace: string, slug: string) {
    return portalApi.getSkill(this, namespace, slug);
  }

  async getNamespace(slug: string) {
    return portalApi.getNamespace(this, slug);
  }

  // ── Frontend page methods ─────────────────────────────────────────────

  async frontendSearch(query: SearchQuery = {}): Promise<Envelope<RegistrySearchReadModel>> {
    return frontendApi.frontendSearch(this, query);
  }

  async frontendSkillDetail(namespace: string, slug: string): Promise<Envelope<SkillDetailReadModel>> {
    return frontendApi.frontendSkillDetail(this, namespace, slug);
  }

  async frontendVersionDetail(namespace: string, slug: string, version: string): Promise<Envelope<VersionDetailReadModel>> {
    return frontendApi.frontendVersionDetail(this, namespace, slug, version);
  }

  async frontendPublishValidate(namespace: string): Promise<Envelope<PublishValidateReadModel>> {
    return frontendApi.frontendPublishValidate(this, namespace);
  }

  async frontendNamespaces(): Promise<Envelope<NamespaceListReadModel>> {
    return frontendApi.frontendNamespaces(this);
  }

  async frontendNamespaceDetail(slug: string): Promise<Envelope<NamespaceDetailReadModel>> {
    return frontendApi.frontendNamespaceDetail(this, slug);
  }

  async frontendReviews(page?: number, size?: number): Promise<Envelope<ReviewQueueReadModel>> {
    return frontendApi.frontendReviews(this, page, size);
  }

  async frontendReviewDetail(id: number): Promise<Envelope<ReviewDetailReadModel>> {
    return frontendApi.frontendReviewDetail(this, id);
  }

  async frontendPromotions(page?: number, size?: number): Promise<Envelope<PromotionQueueReadModel>> {
    return frontendApi.frontendPromotions(this, page, size);
  }

  async frontendPromotionDetail(id: number): Promise<Envelope<PromotionDetailReadModel>> {
    return frontendApi.frontendPromotionDetail(this, id);
  }

  async frontendGovernance(): Promise<Envelope<GovernanceWorkbenchReadModel>> {
    return frontendApi.frontendGovernance(this);
  }

  async frontendAdmin(): Promise<Envelope<AdminPageReadModel>> {
    return frontendApi.frontendAdmin(this);
  }

  // ── Tool API methods ──────────────────────────────────────────────────

  async toolWorkspaceMetadata(): Promise<Envelope<WorkspaceMetadataResponse>> {
    return toolingApi.toolWorkspaceMetadata(this);
  }

  async toolPackageHash(entries: PackageEntry[]): Promise<Envelope<PackageHashResponse>> {
    return toolingApi.toolPackageHash(this, entries);
  }

  async toolResolve(namespace: string, slug: string, version?: string): Promise<Envelope<ResolveResult>> {
    return toolingApi.toolResolve(this, namespace, slug, version);
  }

  async toolInstall(namespace: string, slug: string, version?: string): Promise<Envelope<InstallTarget>> {
    return toolingApi.toolInstall(this, namespace, slug, version);
  }

  async toolDiff(namespace: string, slug: string, fromVersion: string, toVersion: string): Promise<Envelope<VersionDiff>> {
    return toolingApi.toolDiff(this, namespace, slug, fromVersion, toVersion);
  }

  async toolValidate(namespace: string, zipFile: Blob): Promise<Envelope<ToolValidateResponse>> {
    return toolingApi.toolValidate(this, namespace, zipFile);
  }

  async toolPublish(namespace: string, zipFile: Blob): Promise<Envelope<ToolPublishResponse>> {
    return toolingApi.toolPublish(this, namespace, zipFile);
  }

  async toolEvaluate(req: EvaluateRequest): Promise<Envelope<EvaluateResponse>> {
    return toolingApi.toolEvaluate(this, req);
  }

  async toolPropose(req: ProposalRequest): Promise<Envelope<ProposalResponse>> {
    return toolingApi.toolPropose(this, req);
  }

  // ── Release methods ───────────────────────────────────────────────────

  async listReleases(namespace: string, slug: string, page?: number, size?: number): Promise<Envelope<ReleaseListResult>> {
    return releasesApi.listReleases(this, namespace, slug, page, size);
  }

  async getLatestRelease(namespace: string, slug: string, channel?: string): Promise<Envelope<Release>> {
    return releasesApi.getLatestRelease(this, namespace, slug, channel);
  }

  async getRelease(namespace: string, slug: string, releaseId: number): Promise<Envelope<ReleaseDetailResponse>> {
    return releasesApi.getRelease(this, namespace, slug, releaseId);
  }

  async createRelease(namespace: string, slug: string, req: CreateReleaseRequest): Promise<Envelope<Release>> {
    return releasesApi.createRelease(this, namespace, slug, req);
  }

  async updateRelease(namespace: string, slug: string, releaseId: number, req: UpdateReleaseRequest): Promise<Envelope<Release>> {
    return releasesApi.updateRelease(this, namespace, slug, releaseId, req);
  }

  async deleteRelease(namespace: string, slug: string, releaseId: number): Promise<Envelope<{ status: string }>> {
    return releasesApi.deleteRelease(this, namespace, slug, releaseId);
  }

  async publishRelease(namespace: string, slug: string, releaseId: number): Promise<Envelope<Release>> {
    return releasesApi.publishRelease(this, namespace, slug, releaseId);
  }

  // ── Frontend release page methods ─────────────────────────────────────

  async frontendReleaseList(namespace: string, slug: string): Promise<Envelope<ReleaseListReadModel>> {
    return frontendApi.frontendReleaseList(this, namespace, slug);
  }

  async frontendReleaseDetail(namespace: string, slug: string, releaseId: number): Promise<Envelope<ReleaseDetailReadModel>> {
    return frontendApi.frontendReleaseDetail(this, namespace, slug, releaseId);
  }

  // ── Community portal methods ──────────────────────────────────────────

  async listIssues(namespace: string, slug: string, params?: { status?: string; page?: number; size?: number }) {
    return communityApi.listIssues(this, namespace, slug, params);
  }

  async getIssue(namespace: string, slug: string, issueId: number) {
    return communityApi.getIssue(this, namespace, slug, issueId);
  }

  async createIssue(namespace: string, slug: string, body: CreateIssueRequest) {
    return communityApi.createIssue(this, namespace, slug, body);
  }

  async updateIssue(namespace: string, slug: string, issueId: number, body: UpdateIssueRequest) {
    return communityApi.updateIssue(this, namespace, slug, issueId, body);
  }

  async deleteIssue(namespace: string, slug: string, issueId: number) {
    return communityApi.deleteIssue(this, namespace, slug, issueId);
  }

  async listIssueComments(namespace: string, slug: string, issueId: number) {
    return communityApi.listIssueComments(this, namespace, slug, issueId);
  }

  async addIssueComment(namespace: string, slug: string, issueId: number, body: { body: string }) {
    return communityApi.addIssueComment(this, namespace, slug, issueId, body);
  }

  async listDiscussions(namespace: string, slug: string, params?: { category?: string; page?: number; size?: number }) {
    return communityApi.listDiscussions(this, namespace, slug, params);
  }

  async getDiscussion(namespace: string, slug: string, discussionId: number) {
    return communityApi.getDiscussion(this, namespace, slug, discussionId);
  }

  async createDiscussion(namespace: string, slug: string, body: { title: string; body?: string; category?: string }) {
    return communityApi.createDiscussion(this, namespace, slug, body);
  }

  async updateDiscussion(namespace: string, slug: string, discussionId: number, body: Record<string, unknown>) {
    return communityApi.updateDiscussion(this, namespace, slug, discussionId, body);
  }

  async deleteDiscussion(namespace: string, slug: string, discussionId: number) {
    return communityApi.deleteDiscussion(this, namespace, slug, discussionId);
  }

  async listDiscussionComments(namespace: string, slug: string, discussionId: number) {
    return communityApi.listDiscussionComments(this, namespace, slug, discussionId);
  }

  async addDiscussionComment(namespace: string, slug: string, discussionId: number, body: { body: string }) {
    return communityApi.addDiscussionComment(this, namespace, slug, discussionId, body);
  }

  async acceptAnswer(namespace: string, slug: string, discussionId: number, commentId: number) {
    return communityApi.acceptAnswer(this, namespace, slug, discussionId, commentId);
  }

  async listWikiPages(namespace: string, slug: string) {
    return communityApi.listWikiPages(this, namespace, slug);
  }

  async getWikiPage(namespace: string, slug: string, pageSlug: string) {
    return communityApi.getWikiPage(this, namespace, slug, pageSlug);
  }

  async createWikiPage(namespace: string, slug: string, body: { title: string; slug: string; body: string; changeSummary?: string }) {
    return communityApi.createWikiPage(this, namespace, slug, body);
  }

  async updateWikiPage(namespace: string, slug: string, pageSlug: string, body: { body: string; changeSummary?: string }) {
    return communityApi.updateWikiPage(this, namespace, slug, pageSlug, body);
  }

  async listWikiVersions(namespace: string, slug: string, pageSlug: string) {
    return communityApi.listWikiVersions(this, namespace, slug, pageSlug);
  }

  async listProposals(namespace: string, slug: string, params?: { status?: string; page?: number; size?: number }) {
    return communityApi.listProposals(this, namespace, slug, params);
  }

  async getProposal(namespace: string, slug: string, proposalId: number) {
    return communityApi.getProposal(this, namespace, slug, proposalId);
  }

  async createProposal(namespace: string, slug: string, body: { title: string; summary?: string; proposedChangesJSON?: string }) {
    return communityApi.createProposal(this, namespace, slug, body);
  }

  async updateProposal(namespace: string, slug: string, proposalId: number, body: { status?: string; comment?: string }) {
    return communityApi.updateProposal(this, namespace, slug, proposalId, body);
  }

  async communitySearch(namespace: string, slug: string, params?: { query?: string; types?: string; page?: number; size?: number }): Promise<Envelope<CommunitySearchResult>> {
    return communityApi.communitySearch(this, namespace, slug, params);
  }

  // ── Agent CI methods ─────────────────────────────────────────────────

  async listPipelineRuns(skillId: number, page?: number, size?: number): Promise<Envelope<PipelineRunListResult>> {
    return agentciApi.listPipelineRuns(this, skillId, page, size);
  }

  async getPipelineRun(skillId: number, runId: number): Promise<Envelope<PipelineRun>> {
    return agentciApi.getPipelineRun(this, skillId, runId);
  }

  async listCheckRuns(skillId: number, runId: number): Promise<Envelope<CheckRun[]>> {
    return agentciApi.listCheckRuns(this, skillId, runId);
  }

  async getCheckRun(skillId: number, checkId: number): Promise<Envelope<CheckRun>> {
    return agentciApi.getCheckRun(this, skillId, checkId);
  }

  async listCheckArtifacts(skillId: number, checkId: number): Promise<Envelope<CheckArtifact[]>> {
    return agentciApi.listCheckArtifacts(this, skillId, checkId);
  }

  async evaluateGates(skillId: number, params?: { trigger?: string; versionId?: number; releaseId?: number }): Promise<Envelope<GateEvalResult>> {
    return agentciApi.evaluateGates(this, skillId, params);
  }

  // ── Community frontend methods ──────────────────────────────────────────

  async frontendIssueList(namespace: string, slug: string, page?: number, size?: number): Promise<Envelope<IssueListReadModel>> {
    return frontendApi.frontendIssueList(this, namespace, slug, page, size);
  }

  async frontendIssueDetail(namespace: string, slug: string, issueId: number): Promise<Envelope<IssueDetailReadModel>> {
    return frontendApi.frontendIssueDetail(this, namespace, slug, issueId);
  }

  async frontendDiscussionList(namespace: string, slug: string, page?: number, size?: number): Promise<Envelope<DiscussionListReadModel>> {
    return frontendApi.frontendDiscussionList(this, namespace, slug, page, size);
  }

  async frontendDiscussionDetail(namespace: string, slug: string, discussionId: number): Promise<Envelope<DiscussionDetailReadModel>> {
    return frontendApi.frontendDiscussionDetail(this, namespace, slug, discussionId);
  }

  async frontendWikiList(namespace: string, slug: string): Promise<Envelope<WikiPageListReadModel>> {
    return frontendApi.frontendWikiList(this, namespace, slug);
  }

  async frontendWikiDetail(namespace: string, slug: string, pageSlug: string): Promise<Envelope<WikiPageDetailReadModel>> {
    return frontendApi.frontendWikiDetail(this, namespace, slug, pageSlug);
  }

  async frontendProposalList(namespace: string, slug: string, page?: number, size?: number): Promise<Envelope<ProposalListReadModel>> {
    return frontendApi.frontendProposalList(this, namespace, slug, page, size);
  }

  async frontendProposalDetail(namespace: string, slug: string, proposalId: number): Promise<Envelope<ProposalDetailReadModel>> {
    return frontendApi.frontendProposalDetail(this, namespace, slug, proposalId);
  }

  // ── Review/Promotion mutation methods ─────────────────────────────────

  async approveReview(id: number, req?: ReviewMutationRequest): Promise<Envelope<ReviewMutationResponse>> {
    return reviewApi.approveReview(this, id, req);
  }

  async rejectReview(id: number, req?: ReviewMutationRequest): Promise<Envelope<ReviewMutationResponse>> {
    return reviewApi.rejectReview(this, id, req);
  }

  async withdrawReview(id: number): Promise<Envelope<WithdrawResponse>> {
    return reviewApi.withdrawReview(this, id);
  }

  async approvePromotion(id: number, req?: PromotionMutationRequest): Promise<Envelope<PromotionMutationResponse>> {
    return promotionApi.approvePromotion(this, id, req);
  }

  async rejectPromotion(id: number, req?: PromotionMutationRequest): Promise<Envelope<PromotionMutationResponse>> {
    return promotionApi.rejectPromotion(this, id, req);
  }

  async withdrawPromotion(id: number): Promise<Envelope<WithdrawResponse>> {
    return promotionApi.withdrawPromotion(this, id);
  }

  // ── Pagination iterators ──────────────────────────────────────────────

  async *iterFrontendSearch(
    query: SearchQuery & PageIteratorOptions = {},
  ): AsyncGenerator<RegistrySearchReadModel> {
    let page = query.page ?? 0;
    const size = query.size ?? 20;
    const maxPages = query.maxPages ?? 10;
    for (let i = 0; i < maxPages; i++) {
      const result = await this.frontendSearch({ ...query, page, size });
      if (!result.success || !result.data) return;
      yield result.data;
      if (result.data.searchResult.skillIds.length === 0) return;
      if (result.data.searchResult.total <= (page + 1) * size) return;
      page++;
    }
  }

  async *iterFrontendReviews(
    options?: PageIteratorOptions,
  ): AsyncGenerator<ReviewQueueReadModel> {
    let page = options?.page ?? 0;
    const size = options?.size ?? 20;
    const maxPages = options?.maxPages ?? 10;
    for (let i = 0; i < maxPages; i++) {
      const result = await this.frontendReviews(page, size);
      if (!result.success || !result.data) return;
      yield result.data;
      if (!result.data.hasMore) return;
      if (result.data.tasks.length === 0) return;
      page++;
    }
  }

  async *iterFrontendPromotions(
    options?: PageIteratorOptions,
  ): AsyncGenerator<PromotionQueueReadModel> {
    let page = options?.page ?? 0;
    const size = options?.size ?? 20;
    const maxPages = options?.maxPages ?? 10;
    for (let i = 0; i < maxPages; i++) {
      const result = await this.frontendPromotions(page, size);
      if (!result.success || !result.data) return;
      yield result.data;
      if (!result.data.hasMore) return;
      if (result.data.requests.length === 0) return;
      page++;
    }
  }

  async *iterFrontendIssues(
    namespace: string,
    slug: string,
    options?: PageIteratorOptions,
  ): AsyncGenerator<IssueListReadModel> {
    let page = options?.page ?? 0;
    const size = options?.size ?? 20;
    const maxPages = options?.maxPages ?? 10;
    for (let i = 0; i < maxPages; i++) {
      const result = await this.frontendIssueList(namespace, slug, page, size);
      if (!result.success || !result.data) return;
      yield result.data;
      if (result.data.issues.length === 0) return;
      if (result.data.totalCount <= (page + 1) * size) return;
      page++;
    }
  }

  async *iterFrontendDiscussions(
    namespace: string,
    slug: string,
    options?: PageIteratorOptions,
  ): AsyncGenerator<DiscussionListReadModel> {
    let page = options?.page ?? 0;
    const size = options?.size ?? 20;
    const maxPages = options?.maxPages ?? 10;
    for (let i = 0; i < maxPages; i++) {
      const result = await this.frontendDiscussionList(namespace, slug, page, size);
      if (!result.success || !result.data) return;
      yield result.data;
      if (result.data.discussions.length === 0) return;
      if (result.data.totalCount <= (page + 1) * size) return;
      page++;
    }
  }

  async *iterFrontendProposals(
    namespace: string,
    slug: string,
    options?: PageIteratorOptions,
  ): AsyncGenerator<ProposalListReadModel> {
    let page = options?.page ?? 0;
    const size = options?.size ?? 20;
    const maxPages = options?.maxPages ?? 10;
    for (let i = 0; i < maxPages; i++) {
      const result = await this.frontendProposalList(namespace, slug, page, size);
      if (!result.success || !result.data) return;
      yield result.data;
      if (result.data.proposals.length === 0) return;
      if (result.data.totalCount <= (page + 1) * size) return;
      page++;
    }
  }

  async *iterReleases(
    namespace: string,
    slug: string,
    options?: PageIteratorOptions,
  ): AsyncGenerator<ReleaseListResult> {
    let page = options?.page ?? 0;
    const size = options?.size ?? 20;
    const maxPages = options?.maxPages ?? 10;
    for (let i = 0; i < maxPages; i++) {
      const result = await this.listReleases(namespace, slug, page, size);
      if (!result.success || !result.data) return;
      yield result.data;
      if (result.data.releases.length === 0) return;
      if (result.data.totalCount <= (page + 1) * size) return;
      page++;
    }
  }

  async *iterPipelineRuns(
    skillId: number,
    options?: PageIteratorOptions,
  ): AsyncGenerator<PipelineRunListResult> {
    let page = options?.page ?? 0;
    const size = options?.size ?? 20;
    const maxPages = options?.maxPages ?? 10;
    for (let i = 0; i < maxPages; i++) {
      const result = await this.listPipelineRuns(skillId, page, size);
      if (!result.success || !result.data) return;
      yield result.data;
      if (result.data.runs.length === 0) return;
      if (result.data.totalCount <= (page + 1) * size) return;
      page++;
    }
  }
}

export default SkillHubClient;
