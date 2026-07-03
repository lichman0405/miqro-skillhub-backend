/**
 * SkillHub TypeScript client.
 *
 * This client mirrors the Go HTTP route surfaces and provides
 * strongly-typed request/response shapes for frontend consumers.
 */

// ── Configuration ───────────────────────────────────────────────────────

/** Options for constructing a SkillHubClient. */
export interface SkillHubClientOptions {
  /** Base URL of the SkillHub server (default: http://localhost:8080). */
  baseUrl?: string;
  /** Custom fetch implementation (for SSR, tests, or polyfills). */
  fetch?: typeof fetch;
  /** Request credentials mode (e.g. "include" for cookie-based auth). */
  credentials?: RequestCredentials;
  /** Static bearer token sent as Authorization: Bearer <token>. */
  token?: string;
  /** Dynamic token provider called before each request. */
  getToken?: () => string | undefined | Promise<string | undefined>;
  /** Additional headers merged into every request. */
  headers?: HeadersInit;
}

/** Constructor argument: a base URL string or an options object. */
export type SkillHubClientConfig = string | SkillHubClientOptions;

// ── Error handling ──────────────────────────────────────────────────────

/** Typed error thrown by unwrap() and carried by failed envelopes. */
export class SkillHubError extends Error {
  readonly code: string;
  readonly status?: number;
  readonly details?: unknown;
  readonly response?: Response;

  constructor(
    code: string,
    message: string,
    status?: number,
    details?: unknown,
    response?: Response,
  ) {
    super(message);
    this.name = "SkillHubError";
    this.code = code;
    this.status = status;
    this.details = details;
    this.response = response;
  }
}

// ── Pagination ──────────────────────────────────────────────────────────

/** Options for bounded async iteration over paginated endpoints. */
export interface PageIteratorOptions {
  /** Starting page (default 0). */
  page?: number;
  /** Page size (default 20, backend-capped at 100). */
  size?: number;
  /** Maximum pages to fetch before stopping (default 10). */
  maxPages?: number;
}

// ── Envelope ────────────────────────────────────────────────────────────

/** Standard response envelope shared by every SkillHub HTTP endpoint. */
export interface Envelope<T = unknown> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string; details?: unknown };
  /** HTTP status code (attached by the client after each request). */
  status?: number;
}

/** Platform principal returned by auth endpoints. */
export interface Principal {
  userID: string;
  displayName: string;
  email: string;
  authMethod: string;
  platformRoles: Record<string, boolean>;
  isAuthenticated: boolean;
}

/** Search query parameters. */
export interface SearchQuery {
  keyword?: string;
  sortBy?: "relevance" | "downloads" | "rating" | "newest";
  page?: number;
  size?: number;
  labelSlugs?: string[];
  installableOnly?: boolean;
}

/** Search result returned by /api/v1/search. */
export interface SearchResult {
  skillIds: number[];
  total: number;
  page: number;
  size: number;
}

/** Skill detail returned by /api/v1/skills/{ns}/{slug}. */
export interface SkillDetail {
  id: number;
  slug: string;
  displayName: string;
  ownerId: string;
  summary: string;
  visibility: string;
  status: string;
  downloadCount: number;
  starCount: number;
  ratingAvg: number;
  canManage: boolean;
}

// ── Frontend read-model types ──────────────────────────────────────────

/** Available actions for the registry search/home page. */
export interface RegistrySearchActions {
  canCreateSkill: boolean;
  canCreateNamespace: boolean;
  canAccessAdmin: boolean;
}

/** Registry search/home read model. */
export interface RegistrySearchReadModel {
  searchResult: SearchResult;
  featuredLabels: string[];
  availableActions: RegistrySearchActions;
}

/** Available actions for the skill detail page. */
export interface SkillDetailActions {
  canEdit: boolean;
  canPublish: boolean;
  canDelete: boolean;
  canSubmitForReview: boolean;
  canRequestPromotion: boolean;
  canStar: boolean;
  canReport: boolean;
  canManage: boolean;
}

/** Skill detail page read model. */
export interface SkillDetailReadModel {
  skill: SkillDetail;
  versions?: VersionDetail[];
  files?: SkillFile[];
  availableActions: SkillDetailActions;
}

/** A skill version. */
export interface VersionDetail {
  id: number;
  version: string;
  status: string;
}

/** A file in a skill package. */
export interface SkillFile {
  path: string;
  size: number;
}

/** Available actions for the version detail/compare page. */
export interface VersionActions {
  canCompare: boolean;
  canDownload: boolean;
  canSubmitForReview: boolean;
  canRequestPromotion: boolean;
  canYank: boolean;
  canReview: boolean;
}

/** Version detail/compare read model. */
export interface VersionDetailReadModel {
  version: VersionDetail;
  availableActions: VersionActions;
}

/** Available actions for the publish validate page. */
export interface PublishValidateActions {
  canPublish: boolean;
  canOverrideWarnings: boolean;
}

/** Publish validate read model. */
export interface PublishValidateReadModel {
  valid: boolean;
  warnings: string[];
  errors?: string[];
  metadata?: Record<string, unknown>;
  availableActions: PublishValidateActions;
}

/** A namespace. */
export interface Namespace {
  id: number;
  slug: string;
  displayName: string;
  type: string;
  description: string;
}

/** Available actions for namespace listing. */
export interface NamespaceListActions {
  canCreateNamespace: boolean;
}

/** Namespace list read model. */
export interface NamespaceListReadModel {
  namespaces: Namespace[];
  availableActions: NamespaceListActions;
}

/** A namespace member. */
export interface NamespaceMember {
  namespaceId: number;
  userId: string;
  role: string;
}

/** Available actions for namespace detail page. */
export interface NamespaceDetailActions {
  canEdit: boolean;
  canDelete: boolean;
  canManageMembers: boolean;
  canTransferOwner: boolean;
  canLeave: boolean;
  canJoin: boolean;
}

/** Namespace detail read model. */
export interface NamespaceDetailReadModel {
  namespace: Namespace;
  members?: NamespaceMember[];
  availableActions: NamespaceDetailActions;
}

/** A review task view. */
export interface ReviewTaskView {
  id: number;
  skillVersionId: number;
  skillId?: number;
  namespaceId: number;
  namespaceSlug?: string;
  skillSlug?: string;
  skillName?: string;
  version?: string;
  submittedBy: string;
  status: string;
  submittedAt: string;
  canApprove?: boolean;
  canReject?: boolean;
  canWithdraw?: boolean;
}

/** Available actions for the review queue. */
export interface ReviewQueueActions {
  canReview: boolean;
  canSubmit: boolean;
  canWithdraw: boolean;
}

/** Review queue read model. */
export interface ReviewQueueReadModel {
  tasks: ReviewTaskView[];
  pendingCount: number;
  page: number;
  size: number;
  hasMore: boolean;
  availableActions: ReviewQueueActions;
}

/** Available actions for review detail. */
export interface ReviewDetailActions {
  canApprove: boolean;
  canReject: boolean;
  canWithdraw: boolean;
}

/** Review detail read model. */
export interface ReviewDetailReadModel {
  task: ReviewTaskView;
  skillName: string;
  version: string;
  availableActions: ReviewDetailActions;
}

/** A promotion request view. */
export interface PromotionRequestView {
  id: number;
  sourceSkillId: number;
  sourceSkillSlug?: string;
  sourceSkillName?: string;
  sourceVersionId: number;
  sourceVersion?: string;
  targetNamespaceId: number;
  targetNamespaceSlug?: string;
  targetSkillId?: number | null;
  submittedBy: string;
  status: string;
  submittedAt: string;
  canApprove?: boolean;
  canReject?: boolean;
  canWithdraw?: boolean;
}

/** Available actions for the promotion queue. */
export interface PromotionQueueActions {
  canReview: boolean;
  canSubmit: boolean;
  canWithdraw: boolean;
}

/** Promotion queue read model. */
export interface PromotionQueueReadModel {
  requests: PromotionRequestView[];
  pendingCount: number;
  page: number;
  size: number;
  hasMore: boolean;
  availableActions: PromotionQueueActions;
}

/** Available actions for promotion detail. */
export interface PromotionDetailActions {
  canApprove: boolean;
  canReject: boolean;
  canWithdraw: boolean;
}

/** Promotion detail read model. */
export interface PromotionDetailReadModel {
  request: PromotionRequestView;
  sourceSkillName: string;
  availableActions: PromotionDetailActions;
}

/** Aggregate governance counts. */
export interface GovernanceSummaryView {
  total: number;
  unread: number;
  byCategory: Record<string, number>;
  pendingReviews: number;
  pendingPromotions: number;
}

/** A recent governance activity entry. */
export interface GovernanceActivityView {
  id: number;
  category: string;
  title: string;
  createdAt: string;
  isRead: boolean;
}

/** Available actions for governance workbench. */
export interface GovernanceWorkbenchActions {
  canReview: boolean;
  canAccessAdmin: boolean;
  canViewAuditLog: boolean;
}

/** Governance workbench read model. */
export interface GovernanceWorkbenchReadModel {
  summary?: GovernanceSummaryView;
  recentActivity: GovernanceActivityView[];
  availableActions: GovernanceWorkbenchActions;
}

/** Admin dashboard statistics. */
export interface AdminStatsView {
  totalSkills: number;
  totalNamespaces: number;
  totalUsers: number;
  pendingReviews: number;
  pendingPromotions: number;
  openReports: number;
}

/** Available actions for the admin page. */
export interface AdminPageActions {
  canManageSkills: boolean;
  canManageUsers: boolean;
  canManageLabels: boolean;
  canResolveReports: boolean;
  canRebuildSearch: boolean;
  canViewAuditLog: boolean;
  canManageNamespaces: boolean;
}

/** Admin page read model. */
export interface AdminPageReadModel {
  stats: AdminStatsView;
  availableActions: AdminPageActions;
}

// ── Tool API types (miqro CLI protocol surface) ──────────────────────────

/** Package entry for manifest/hash computation. */
export interface PackageEntry {
  path: string;
  content: string;
  size: number;
  contentType: string;
}

/** A single entry in a deterministic package manifest. */
export interface ManifestEntry {
  path: string;
  size: number;
  contentType: string;
  sha256: string;
}

/** Deterministic package manifest. */
export interface PackageManifest {
  entries: ManifestEntry[];
  hash: string;
  totalSize: number;
  fileCount: number;
}

/** Request to compute deterministic package hash. */
export interface PackageHashRequest {
  entries: PackageEntry[];
}

/** Response from package hash computation. */
export interface PackageHashResponse {
  manifest: PackageManifest;
}

/** Workspace metadata response (miqro init contract). */
export interface WorkspaceMetadataResponse {
  workspace: {
    requiredFiles: string[];
    optionalFiles: string[];
    manifestFormat: string;
    schema: {
      fields: string[];
      required: string[];
    };
  };
}

/** Resolved version with tooling fingerprint. */
export interface ResolveResult {
  skillId: number;
  namespace: string;
  slug: string;
  version: string;
  versionId: number;
  fingerprint: string;
  downloadUrl: string;
}

/** Agent runtime descriptor. */
export interface AgentRuntime {
  type: string;
  minVersion?: string;
}

/** Install target metadata. */
export interface InstallTarget {
  skillId: number;
  skillSlug: string;
  namespace: string;
  version: string;
  fingerprint: string;
  downloadUrl: string;
  supportedAgents?: AgentRuntime[];
  installPath?: string;
}

/** Diff summary counts. */
export interface DiffSummary {
  totalFiles: number;
  addedFiles: number;
  modifiedFiles: number;
  removedFiles: number;
  addedLines: number;
  removedLines: number;
}

/** A single line in a diff hunk. */
export interface DiffLine {
  type: "ADD" | "DELETE" | "CONTEXT";
  content: string;
  oldLineNumber?: number;
  newLineNumber?: number;
}

/** A contiguous change block. */
export interface DiffHunk {
  oldStart: number;
  oldLines: number;
  newStart: number;
  newLines: number;
  lines: DiffLine[];
}

/** A single file in a version diff. */
export interface DiffFile {
  path: string;
  changeType: "ADDED" | "REMOVED" | "MODIFIED";
  oldSize?: number;
  newSize?: number;
  binary: boolean;
  truncated: boolean;
  hunks?: DiffHunk[];
}

/** Full version diff. */
export interface VersionDiff {
  fromVersion: string;
  toVersion: string;
  summary: DiffSummary;
  files: DiffFile[];
}

/** Tool-facing validation result. */
export interface ToolValidateResponse {
  valid: boolean;
  errors?: string[];
  warnings?: string[];
  resolvedSlug?: string;
  resolvedVersion?: string;
}

/** Tool-facing publish response. */
export interface ToolPublishResponse {
  skillId: number;
  slug: string;
  version: {
    id: number;
    version: string;
    status: string;
  };
}

/** Evaluate trigger request (placeholder). */
export interface EvaluateRequest {
  skillId: number;
  versionId: number;
  trigger: string;
}

/** Evaluate trigger response (placeholder). */
export interface EvaluateResponse {
  accepted: boolean;
  checkRunId?: string;
  message?: string;
}

/** Proposal preparation request (placeholder). */
export interface ProposalRequest {
  skillId: number;
  namespace: string;
  slug: string;
  title: string;
  description: string;
  diffSummary?: VersionDiff;
}

/** Proposal preparation response (placeholder). */
export interface ProposalResponse {
  accepted: boolean;
  proposalId?: string;
  message?: string;
}

// ── Release types ────────────────────────────────────────────────────────

/** A skill release. */
export interface Release {
  id: number;
  skillId: number;
  versionId: number;
  channel: string;
  title: string;
  notes?: string;
  draft: boolean;
  prerelease: boolean;
  yanked: boolean;
  publishedAt?: string;
  publisherId: string;
  reviewerId?: string;
  packageHash?: string;
  ciCheckRunId?: string;
}

/** A release asset. */
export interface ReleaseAsset {
  id: number;
  name: string;
  label?: string;
  contentType: string;
  size: number;
  downloadCount: number;
}

/** Paginated release list. */
export interface ReleaseListResult {
  releases: Release[];
  totalCount: number;
  page: number;
  size: number;
}

/** Release with assets. */
export interface ReleaseDetailResponse {
  release: Release;
  assets: ReleaseAsset[];
}

/** Create release request body. */
export interface CreateReleaseRequest {
  versionId: number;
  channel?: string;
  title: string;
  notes?: string;
  draft?: boolean;
  prerelease?: boolean;
}

/** Update release request body. */
export interface UpdateReleaseRequest {
  title?: string;
  notes?: string;
  draft?: boolean;
  prerelease?: boolean;
  yanked?: boolean;
}

// ── Frontend release read-model types ─────────────────────────────────────

/** Release list read model. */
export interface ReleaseListReadModel {
  releases: ReleaseListView[];
  totalCount: number;
  page: number;
  size: number;
  availableActions: ReleaseListActions;
}

/** Release list view (summary). */
export interface ReleaseListView {
  id: number;
  versionId: number;
  channel: string;
  title: string;
  draft: boolean;
  prerelease: boolean;
  yanked: boolean;
  publishedAt?: string;
  publisherId: string;
}

/** Actions for release list page. */
export interface ReleaseListActions {
  canCreateRelease: boolean;
}

/** Release detail read model. */
export interface ReleaseDetailReadModel {
  release: ReleaseDetailView;
  assets?: ReleaseAssetView[];
  availableActions: ReleaseDetailActions;
}

/** Release detail view. */
export interface ReleaseDetailView {
  id: number;
  skillId: number;
  versionId: number;
  channel: string;
  title: string;
  notes?: string;
  draft: boolean;
  prerelease: boolean;
  yanked: boolean;
  publishedAt?: string;
  publisherId: string;
  reviewerId?: string;
  packageHash?: string;
  ciCheckRunId?: string;
}

/** Release asset view. */
export interface ReleaseAssetView {
  id: number;
  name: string;
  label?: string;
  contentType: string;
  size: number;
  downloadCount: number;
}

/** Actions for release detail page. */
export interface ReleaseDetailActions {
  canEdit: boolean;
  canDelete: boolean;
  canYank: boolean;
  canUnYank: boolean;
}

// ── Community Types ────────────────────────────────────────────────────

/** A skill-scoped issue. */
export interface Issue {
  id: number;
  skillId: number;
  title: string;
  body?: string;
  status: "OPEN" | "CLOSED";
  assigneeId?: string;
  linkedVersionId?: number;
  linkedReleaseId?: number;
  authorId: string;
  locked: boolean;
  commentCount: number;
}

export interface IssueComment {
  id: number;
  issueId: number;
  authorId: string;
  body: string;
}

export interface CreateIssueRequest {
  title: string;
  body?: string;
  assigneeId?: string;
  linkedVersionId?: number;
  linkedReleaseId?: number;
}

export interface UpdateIssueRequest {
  title?: string;
  body?: string;
  status?: "OPEN" | "CLOSED";
  assigneeId?: string;
  locked?: boolean;
}

export interface Discussion {
  id: number;
  skillId: number;
  title: string;
  body?: string;
  category: "GENERAL" | "QA" | "IDEAS" | "ANNOUNCEMENTS";
  acceptedAnswerId?: number;
  authorId: string;
  locked: boolean;
  pinned: boolean;
  commentCount: number;
}

export interface DiscussionComment {
  id: number;
  discussionId: number;
  authorId: string;
  body: string;
  isAnswer: boolean;
}

export interface WikiPage {
  id: number;
  skillId: number;
  title: string;
  slug: string;
  currentVersionId?: number;
  orderIndex: number;
}

export interface WikiPageVersion {
  id: number;
  pageId: number;
  body: string;
  version: number;
  changeSummary?: string;
  authorId: string;
}

export interface ChangeProposal {
  id: number;
  skillId: number;
  title: string;
  summary?: string;
  status: "OPEN" | "ACCEPTED" | "REJECTED" | "WITHDRAWN";
  authorId: string;
  reviewerId?: string;
  sourceGitRef?: string;
  reviewComment?: string;
}

// ── Community Frontend Read Models ─────────────────────────────────────

export interface IssueListView {
  id: number; title: string; status: string; authorId: string; locked: boolean; commentCount: number;
}
export interface IssueListActions { canCreateIssue: boolean; }
export interface IssueListReadModel {
  issues: IssueListView[]; totalCount: number; page: number; size: number; availableActions: IssueListActions;
}
export interface IssueDetailView {
  id: number; title: string; body?: string; status: string; authorId: string; locked: boolean;
}
export interface IssueDetailActions { canEdit: boolean; canDelete: boolean; canClose: boolean; canReopen: boolean; }
export interface IssueDetailReadModel {
  issue: IssueDetailView; comments?: CommentView[]; availableActions: IssueDetailActions;
}
export interface CommentView { id: number; authorId: string; body: string; }

export interface DiscussionListView {
  id: number; title: string; category: string; authorId: string; pinned: boolean; locked: boolean; commentCount: number;
}
export interface DiscussionListActions { canCreateDiscussion: boolean; }
export interface DiscussionListReadModel {
  discussions: DiscussionListView[]; totalCount: number; page: number; size: number; availableActions: DiscussionListActions;
}
export interface DiscussionDetailView {
  id: number; title: string; body?: string; category: string; authorId: string; pinned: boolean; locked: boolean;
}
export interface DiscussionDetailActions { canEdit: boolean; canDelete: boolean; canLock: boolean; canPin: boolean; canAcceptAnswer: boolean; }
export interface DiscussionDetailReadModel {
  discussion: DiscussionDetailView; comments?: CommentView[]; availableActions: DiscussionDetailActions;
}

export interface WikiPageListView { id: number; title: string; slug: string; orderIndex: number; }
export interface WikiPageListActions { canCreatePage: boolean; }
export interface WikiPageListReadModel { pages: WikiPageListView[]; availableActions: WikiPageListActions; }
export interface WikiPageDetailView { id: number; title: string; slug: string; version: number; body?: string; }
export interface WikiVersionView { id: number; version: number; changeSummary?: string; authorId: string; }
export interface WikiPageDetailActions { canEdit: boolean; canDelete: boolean; }
export interface WikiPageDetailReadModel {
  page: WikiPageDetailView; versions?: WikiVersionView[]; availableActions: WikiPageDetailActions;
}

export interface ProposalListView { id: number; title: string; status: string; authorId: string; }
export interface ProposalListActions { canCreateProposal: boolean; }
export interface ProposalListReadModel {
  proposals: ProposalListView[]; totalCount: number; page: number; size: number; availableActions: ProposalListActions;
}
export interface ProposalDetailView {
  id: number; title: string; summary?: string; status: string; authorId: string; reviewerId?: string; sourceGitRef?: string;
}
export interface ProposalDetailActions { canAccept: boolean; canReject: boolean; canWithdraw: boolean; }
export interface ProposalDetailReadModel {
  proposal: ProposalDetailView; availableActions: ProposalDetailActions;
}

// ── Agent CI Types ─────────────────────────────────────────────────────

/** A single CI pipeline run. */
export interface PipelineRun {
  id: number;
  pipelineId: number;
  skillId: number;
  versionId?: number;
  releaseId?: number;
  triggerType: string;
  triggeredBy: string;
  status: string;
  checkCount: number;
  passedCount: number;
  failedCount: number;
  skippedCount: number;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
}

/** Paginated pipeline run list. */
export interface PipelineRunListResult {
  runs: PipelineRun[];
  totalCount: number;
  page: number;
  size: number;
}

/** A single CI check run. */
export interface CheckRun {
  id: number;
  pipelineRunId: number;
  skillId: number;
  versionId?: number;
  releaseId?: number;
  name: string;
  runnerType: string;
  status: string;
  conclusion?: string;
  summary?: string;
  isBlocking: boolean;
  startedAt?: string;
  completedAt?: string;
  durationMs?: number;
  createdAt: string;
  updatedAt: string;
}

/** A CI check artifact. */
export interface CheckArtifact {
  id: number;
  checkRunId: number;
  name: string;
  contentType: string;
  size: number;
  storageKey: string;
  createdAt: string;
}

/** A single gate policy evaluation result. */
export interface GatePolicyResult {
  policyId: number;
  policyName: string;
  passed: boolean;
  reason?: string;
}

/** Gate evaluation result. */
export interface GateEvalResult {
  passed: boolean;
  reason?: string;
  policyResults?: GatePolicyResult[];
}

// ── Community Search ───────────────────────────────────────────────────

export interface CommunitySearchResultItem {
  type: string; id: number; skillId: number; title: string; snippet?: string;
}
export interface CommunitySearchResult {
  items: CommunitySearchResultItem[]; totalCount: number; page: number; size: number;
}

// ── Review/Promotion Mutation Types ────────────────────────────────────

/** Request body for review/promotion approve and reject. */
export interface ReviewMutationRequest {
  comment?: string;
}

/** Response for review approve/reject. */
export interface ReviewMutationResponse {
  task: ReviewTaskView;
}

/** Request body for promotion approve and reject (same shape). */
export interface PromotionMutationRequest {
  comment?: string;
}

/** Response for promotion approve/reject. */
export interface PromotionMutationResponse {
  request: PromotionRequestView;
}

/** Response for review/promotion withdraw. */
export interface WithdrawResponse {
  status: string;
  version?: VersionDetail;
}

// ── Client ─────────────────────────────────────────────────────────────

/** SkillHub API client — thin HTTP wrapper over the backend. */
export class SkillHubClient {
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

  // ── Internal helpers ────────────────────────────────────────────────

  /** Build a URL-safe path from segments, encoding each one. */
  private path(...parts: Array<string | number>): string {
    return parts.map((p) => encodeURIComponent(String(p))).join("/");
  }

  /** Build a query string from a params record. */
  private query(
    params: Record<string, string | number | boolean | string[] | undefined>,
  ): string {
    const sp = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (value === undefined) continue;
      if (Array.isArray(value)) {
        sp.set(key, value.join(","));
      } else {
        sp.set(key, String(value));
      }
    }
    const qs = sp.toString();
    return qs ? `?${qs}` : "";
  }

  /** Execute an HTTP request and return the JSON envelope. */
  private async fetch<T>(
    path: string,
    init?: RequestInit,
  ): Promise<Envelope<T>> {
    // Build headers — start with custom defaults
    const headers = new Headers(this.customHeaders);

    // Set Content-Type: application/json unless body is FormData or caller sets it
    const isFormData = init?.body instanceof FormData;
    const reqHeaders = init?.headers ? new Headers(init.headers) : null;
    const hasContentType =
      headers.has("content-type") || reqHeaders?.has("content-type");
    if (!hasContentType && !isFormData) {
      headers.set("Content-Type", "application/json");
    }

    // Merge per-request headers
    if (reqHeaders) {
      reqHeaders.forEach((v, k) => headers.set(k, v));
    }

    // Auth
    if (this.token) {
      headers.set("Authorization", `Bearer ${this.token}`);
    } else if (this.getToken) {
      const t = await this.getToken();
      if (t) headers.set("Authorization", `Bearer ${t}`);
    }

    // Build fetch init
    const fetchInit: RequestInit = { ...init, headers };

    // Credentials: per-request overrides constructor default
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

    // Attach HTTP status for unwrap() / error inspection
    if (body && typeof body === "object") {
      (body as unknown as Record<string, unknown>).status = res.status;
    }

    return body as unknown as Envelope<T>;
  }

  // ── Unwrap helper ───────────────────────────────────────────────────

  /**
   * Unwrap a promise or envelope, returning the data on success or
   * throwing a SkillHubError on failure.
   */
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

  // ── Auth ────────────────────────────────────────────────────────────

  /** Login with local credentials. */
  async login(username: string, password: string) {
    return this.fetch<Principal>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    });
  }

  /** Get current user. */
  async me(): Promise<Envelope<Principal>> {
    return this.fetch("/api/v1/auth/me", { credentials: "include" });
  }

  // ── Search ──────────────────────────────────────────────────────────

  /** Search skills. */
  async search(query: SearchQuery): Promise<Envelope<SearchResult>> {
    return this.fetch(
      `/api/v1/search${this.query({
        keyword: query.keyword,
        sortBy: query.sortBy,
        page: query.page,
        size: query.size,
        labelSlugs: query.labelSlugs,
        installableOnly: query.installableOnly,
      })}`,
    );
  }

  // ── Portal detail ───────────────────────────────────────────────────

  /** Get a skill by namespace and slug. */
  async getSkill(namespace: string, slug: string) {
    return this.fetch<SkillDetail>(
      `/api/v1/skills/${this.path(namespace, slug)}`,
    );
  }

  /** Get a namespace by slug. */
  async getNamespace(slug: string) {
    return this.fetch(`/api/v1/namespaces/${this.path(slug)}`);
  }

  // ── Frontend page methods ────────────────────────────────────────────

  /** Get registry search/home page read model. */
  async frontendSearch(
    query: SearchQuery = {},
  ): Promise<Envelope<RegistrySearchReadModel>> {
    return this.fetch(
      `/api/v1/frontend/search${this.query({
        q: query.keyword,
        sort: query.sortBy,
        page: query.page,
        size: query.size,
        labels: query.labelSlugs,
        installable: query.installableOnly,
      })}`,
    );
  }

  /** Get skill detail page read model (viewer-scoped). */
  async frontendSkillDetail(
    namespace: string,
    slug: string,
  ): Promise<Envelope<SkillDetailReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}`,
    );
  }

  /** Get version detail/compare page read model. */
  async frontendVersionDetail(
    namespace: string,
    slug: string,
    version: string,
  ): Promise<Envelope<VersionDetailReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}/versions/${this.path(version)}`,
    );
  }

  /** Get publish validate page read model. */
  async frontendPublishValidate(
    namespace: string,
  ): Promise<Envelope<PublishValidateReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace)}/publish/validate`,
    );
  }

  /** Get namespace list page read model. */
  async frontendNamespaces(): Promise<Envelope<NamespaceListReadModel>> {
    return this.fetch("/api/v1/frontend/namespaces");
  }

  /** Get namespace detail page read model (viewer-scoped). */
  async frontendNamespaceDetail(
    slug: string,
  ): Promise<Envelope<NamespaceDetailReadModel>> {
    return this.fetch(`/api/v1/frontend/namespaces/${this.path(slug)}`);
  }

  /** Get review queue page read model. Supports optional page/size query params. */
  async frontendReviews(
    page?: number,
    size?: number,
  ): Promise<Envelope<ReviewQueueReadModel>> {
    return this.fetch(
      `/api/v1/frontend/reviews${this.query({ page, size })}`,
    );
  }

  /** Get review detail page read model. */
  async frontendReviewDetail(
    id: number,
  ): Promise<Envelope<ReviewDetailReadModel>> {
    return this.fetch(`/api/v1/frontend/reviews/${this.path(id)}`);
  }

  /** Get promotion queue page read model. Supports optional page/size query params. */
  async frontendPromotions(
    page?: number,
    size?: number,
  ): Promise<Envelope<PromotionQueueReadModel>> {
    return this.fetch(
      `/api/v1/frontend/promotions${this.query({ page, size })}`,
    );
  }

  /** Get promotion detail page read model. */
  async frontendPromotionDetail(
    id: number,
  ): Promise<Envelope<PromotionDetailReadModel>> {
    return this.fetch(`/api/v1/frontend/promotions/${this.path(id)}`);
  }

  /** Get governance workbench page read model. */
  async frontendGovernance(): Promise<Envelope<GovernanceWorkbenchReadModel>> {
    return this.fetch("/api/v1/frontend/governance");
  }

  /** Get admin page read model. */
  async frontendAdmin(): Promise<Envelope<AdminPageReadModel>> {
    return this.fetch("/api/v1/frontend/admin");
  }

  // ── Tool API methods (miqro CLI protocol surface) ──────────────────────

  /** Get workspace metadata contract (miqro init). */
  async toolWorkspaceMetadata(): Promise<Envelope<WorkspaceMetadataResponse>> {
    return this.fetch("/api/tool/v1/workspace/metadata");
  }

  /** Compute deterministic package hash (miqro pack). */
  async toolPackageHash(
    entries: PackageEntry[],
  ): Promise<Envelope<PackageHashResponse>> {
    return this.fetch("/api/tool/v1/packages/hash", {
      method: "POST",
      body: JSON.stringify({ entries }),
    });
  }

  /** Resolve a skill version with fingerprint (miqro resolve). */
  async toolResolve(
    namespace: string,
    slug: string,
    version?: string,
  ): Promise<Envelope<ResolveResult>> {
    return this.fetch(
      `/api/tool/v1/skills/${this.path(namespace, slug)}/resolve${this.query({ version })}`,
    );
  }

  /** Get install target metadata (miqro install). */
  async toolInstall(
    namespace: string,
    slug: string,
    version?: string,
  ): Promise<Envelope<InstallTarget>> {
    return this.fetch(
      `/api/tool/v1/skills/${this.path(namespace, slug)}/install${this.query({ version })}`,
    );
  }

  /** Diff two skill versions (miqro diff). */
  async toolDiff(
    namespace: string,
    slug: string,
    fromVersion: string,
    toVersion: string,
  ): Promise<Envelope<VersionDiff>> {
    return this.fetch(
      `/api/tool/v1/skills/${this.path(namespace, slug)}/diff${this.query({ from: fromVersion, to: toVersion })}`,
    );
  }

  /** Validate a skill package (miqro validate). Accepts a zip File/Blob. */
  async toolValidate(
    namespace: string,
    zipFile: Blob,
  ): Promise<Envelope<ToolValidateResponse>> {
    const formData = new FormData();
    formData.append("package", zipFile);
    return this.fetch(
      `/api/tool/v1/skills/${this.path(namespace)}/validate`,
      {
        method: "POST",
        body: formData,
        headers: {}, // let browser set multipart Content-Type
      },
    );
  }

  /** Publish a skill package (miqro publish). Accepts a zip File/Blob. */
  async toolPublish(
    namespace: string,
    zipFile: Blob,
  ): Promise<Envelope<ToolPublishResponse>> {
    const formData = new FormData();
    formData.append("package", zipFile);
    return this.fetch(
      `/api/tool/v1/skills/${this.path(namespace)}/publish`,
      {
        method: "POST",
        body: formData,
        headers: {}, // let browser set multipart Content-Type
      },
    );
  }

  /** Trigger skill evaluation (miqro evaluate — Phase 12 placeholder). */
  async toolEvaluate(
    req: EvaluateRequest,
  ): Promise<Envelope<EvaluateResponse>> {
    return this.fetch("/api/tool/v1/evaluate/trigger", {
      method: "POST",
      body: JSON.stringify(req),
    });
  }

  /** Prepare a skill change proposal (miqro propose — Phase 11 placeholder). */
  async toolPropose(
    req: ProposalRequest,
  ): Promise<Envelope<ProposalResponse>> {
    return this.fetch("/api/tool/v1/proposals/prepare", {
      method: "POST",
      body: JSON.stringify(req),
    });
  }

  // ── Release methods ────────────────────────────────────────────────────

  /** List releases for a skill. */
  async listReleases(
    namespace: string,
    slug: string,
    page?: number,
    size?: number,
  ): Promise<Envelope<ReleaseListResult>> {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/releases${this.query({ page, size })}`,
    );
  }

  /** Get latest stable release for a skill. */
  async getLatestRelease(
    namespace: string,
    slug: string,
    channel?: string,
  ): Promise<Envelope<Release>> {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/releases/latest${this.query({ channel })}`,
    );
  }

  /** Get a single release by ID. */
  async getRelease(
    namespace: string,
    slug: string,
    releaseId: number,
  ): Promise<Envelope<ReleaseDetailResponse>> {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/releases/${this.path(releaseId)}`,
    );
  }

  /** Create a new release. */
  async createRelease(
    namespace: string,
    slug: string,
    req: CreateReleaseRequest,
  ): Promise<Envelope<Release>> {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/releases`,
      {
        method: "POST",
        body: JSON.stringify(req),
      },
    );
  }

  /** Update release metadata. */
  async updateRelease(
    namespace: string,
    slug: string,
    releaseId: number,
    req: UpdateReleaseRequest,
  ): Promise<Envelope<Release>> {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/releases/${this.path(releaseId)}`,
      { method: "PATCH", body: JSON.stringify(req) },
    );
  }

  /** Delete a release. */
  async deleteRelease(
    namespace: string,
    slug: string,
    releaseId: number,
  ): Promise<Envelope<{ status: string }>> {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/releases/${this.path(releaseId)}`,
      { method: "DELETE" },
    );
  }

  /** Publish a draft release (runs gate enforcement). */
  async publishRelease(
    namespace: string,
    slug: string,
    releaseId: number,
  ): Promise<Envelope<Release>> {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/releases/${this.path(releaseId)}/publish`,
      { method: "POST" },
    );
  }

  // ── Frontend release page methods ──────────────────────────────────────

  /** Get release list page read model. */
  async frontendReleaseList(
    namespace: string,
    slug: string,
  ): Promise<Envelope<ReleaseListReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}/releases`,
    );
  }

  /** Get release detail page read model. */
  async frontendReleaseDetail(
    namespace: string,
    slug: string,
    releaseId: number,
  ): Promise<Envelope<ReleaseDetailReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}/releases/${this.path(releaseId)}`,
    );
  }

  // ── Community portal methods ──────────────────────────────────────

  async listIssues(
    namespace: string,
    slug: string,
    params?: { status?: string; page?: number; size?: number },
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/issues${this.query({
        status: params?.status,
        page: params?.page,
        size: params?.size,
      })}`,
    );
  }

  async getIssue(namespace: string, slug: string, issueId: number) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/issues/${this.path(issueId)}`,
    );
  }

  async createIssue(
    namespace: string,
    slug: string,
    body: CreateIssueRequest,
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/issues`,
      {
        method: "POST",
        body: JSON.stringify(body),
      },
    );
  }

  async updateIssue(
    namespace: string,
    slug: string,
    issueId: number,
    body: UpdateIssueRequest,
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/issues/${this.path(issueId)}`,
      {
        method: "PATCH",
        body: JSON.stringify(body),
      },
    );
  }

  async deleteIssue(namespace: string, slug: string, issueId: number) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/issues/${this.path(issueId)}`,
      { method: "DELETE" },
    );
  }

  async listIssueComments(
    namespace: string,
    slug: string,
    issueId: number,
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/issues/${this.path(issueId)}/comments`,
    );
  }

  async addIssueComment(
    namespace: string,
    slug: string,
    issueId: number,
    body: { body: string },
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/issues/${this.path(issueId)}/comments`,
      {
        method: "POST",
        body: JSON.stringify(body),
      },
    );
  }

  async listDiscussions(
    namespace: string,
    slug: string,
    params?: { category?: string; page?: number; size?: number },
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/discussions${this.query({
        category: params?.category,
        page: params?.page,
        size: params?.size,
      })}`,
    );
  }

  async getDiscussion(
    namespace: string,
    slug: string,
    discussionId: number,
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/discussions/${this.path(discussionId)}`,
    );
  }

  async createDiscussion(
    namespace: string,
    slug: string,
    body: { title: string; body?: string; category?: string },
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/discussions`,
      {
        method: "POST",
        body: JSON.stringify(body),
      },
    );
  }

  async updateDiscussion(
    namespace: string,
    slug: string,
    discussionId: number,
    body: Record<string, unknown>,
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/discussions/${this.path(discussionId)}`,
      {
        method: "PATCH",
        body: JSON.stringify(body),
      },
    );
  }

  async deleteDiscussion(
    namespace: string,
    slug: string,
    discussionId: number,
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/discussions/${this.path(discussionId)}`,
      { method: "DELETE" },
    );
  }

  async listDiscussionComments(
    namespace: string,
    slug: string,
    discussionId: number,
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/discussions/${this.path(discussionId)}/comments`,
    );
  }

  async addDiscussionComment(
    namespace: string,
    slug: string,
    discussionId: number,
    body: { body: string },
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/discussions/${this.path(discussionId)}/comments`,
      {
        method: "POST",
        body: JSON.stringify(body),
      },
    );
  }

  async acceptAnswer(
    namespace: string,
    slug: string,
    discussionId: number,
    commentId: number,
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/discussions/${this.path(discussionId)}/accept-answer`,
      {
        method: "POST",
        body: JSON.stringify({ commentId }),
      },
    );
  }

  async listWikiPages(namespace: string, slug: string) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/wiki`,
    );
  }

  async getWikiPage(namespace: string, slug: string, pageSlug: string) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/wiki/${this.path(pageSlug)}`,
    );
  }

  async createWikiPage(
    namespace: string,
    slug: string,
    body: {
      title: string;
      slug: string;
      body: string;
      changeSummary?: string;
    },
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/wiki`,
      {
        method: "POST",
        body: JSON.stringify(body),
      },
    );
  }

  async updateWikiPage(
    namespace: string,
    slug: string,
    pageSlug: string,
    body: { body: string; changeSummary?: string },
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/wiki/${this.path(pageSlug)}`,
      {
        method: "PUT",
        body: JSON.stringify(body),
      },
    );
  }

  async listWikiVersions(
    namespace: string,
    slug: string,
    pageSlug: string,
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/wiki/${this.path(pageSlug)}/versions`,
    );
  }

  async listProposals(
    namespace: string,
    slug: string,
    params?: { status?: string; page?: number; size?: number },
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/proposals${this.query({
        status: params?.status,
        page: params?.page,
        size: params?.size,
      })}`,
    );
  }

  async getProposal(
    namespace: string,
    slug: string,
    proposalId: number,
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/proposals/${this.path(proposalId)}`,
    );
  }

  async createProposal(
    namespace: string,
    slug: string,
    body: {
      title: string;
      summary?: string;
      proposedChangesJSON?: string;
    },
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/proposals`,
      {
        method: "POST",
        body: JSON.stringify(body),
      },
    );
  }

  async updateProposal(
    namespace: string,
    slug: string,
    proposalId: number,
    body: { status?: string; comment?: string },
  ) {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/proposals/${this.path(proposalId)}`,
      {
        method: "PATCH",
        body: JSON.stringify(body),
      },
    );
  }

  // ── Community search ─────────────────────────────────────────────

  async communitySearch(
    namespace: string,
    slug: string,
    params?: {
      query?: string;
      types?: string;
      page?: number;
      size?: number;
    },
  ): Promise<Envelope<CommunitySearchResult>> {
    return this.fetch(
      `/api/v1/skills/${this.path(namespace, slug)}/community/search${this.query({
        query: params?.query,
        types: params?.types,
        page: params?.page,
        size: params?.size,
      })}`,
    );
  }

  // ── Agent CI methods ─────────────────────────────────────────────

  /** List CI pipeline runs for a skill. */
  async listPipelineRuns(
    skillId: number,
    page?: number,
    size?: number,
  ): Promise<Envelope<PipelineRunListResult>> {
    return this.fetch(
      `/api/v1/skills/${this.path(skillId)}/ci/runs${this.query({ page, size })}`,
    );
  }

  /** Get a single pipeline run. */
  async getPipelineRun(
    skillId: number,
    runId: number,
  ): Promise<Envelope<PipelineRun>> {
    return this.fetch(
      `/api/v1/skills/${this.path(skillId)}/ci/runs/${this.path(runId)}`,
    );
  }

  /** List check runs for a pipeline run. */
  async listCheckRuns(
    skillId: number,
    runId: number,
  ): Promise<Envelope<CheckRun[]>> {
    return this.fetch(
      `/api/v1/skills/${this.path(skillId)}/ci/runs/${this.path(runId)}/checks`,
    );
  }

  /** Get a single check run. */
  async getCheckRun(
    skillId: number,
    checkId: number,
  ): Promise<Envelope<CheckRun>> {
    return this.fetch(
      `/api/v1/skills/${this.path(skillId)}/ci/checks/${this.path(checkId)}`,
    );
  }

  /** List artifacts for a check run. */
  async listCheckArtifacts(
    skillId: number,
    checkId: number,
  ): Promise<Envelope<CheckArtifact[]>> {
    return this.fetch(
      `/api/v1/skills/${this.path(skillId)}/ci/checks/${this.path(checkId)}/artifacts`,
    );
  }

  /** Evaluate CI gates for a skill. */
  async evaluateGates(
    skillId: number,
    params?: { trigger?: string; versionId?: number; releaseId?: number },
  ): Promise<Envelope<GateEvalResult>> {
    return this.fetch(
      `/api/v1/skills/${this.path(skillId)}/ci/gates${this.query({
        trigger: params?.trigger,
        versionId: params?.versionId,
        releaseId: params?.releaseId,
      })}`,
    );
  }

  // ── Community frontend methods ────────────────────────────────────

  async frontendIssueList(
    namespace: string,
    slug: string,
    page?: number,
    size?: number,
  ): Promise<Envelope<IssueListReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}/issues${this.query({ page, size })}`,
    );
  }

  async frontendIssueDetail(
    namespace: string,
    slug: string,
    issueId: number,
  ): Promise<Envelope<IssueDetailReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}/issues/${this.path(issueId)}`,
    );
  }

  async frontendDiscussionList(
    namespace: string,
    slug: string,
    page?: number,
    size?: number,
  ): Promise<Envelope<DiscussionListReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}/discussions${this.query({ page, size })}`,
    );
  }

  async frontendDiscussionDetail(
    namespace: string,
    slug: string,
    discussionId: number,
  ): Promise<Envelope<DiscussionDetailReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}/discussions/${this.path(discussionId)}`,
    );
  }

  async frontendWikiList(
    namespace: string,
    slug: string,
  ): Promise<Envelope<WikiPageListReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}/wiki`,
    );
  }

  async frontendWikiDetail(
    namespace: string,
    slug: string,
    pageSlug: string,
  ): Promise<Envelope<WikiPageDetailReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}/wiki/${this.path(pageSlug)}`,
    );
  }

  async frontendProposalList(
    namespace: string,
    slug: string,
    page?: number,
    size?: number,
  ): Promise<Envelope<ProposalListReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}/proposals${this.query({ page, size })}`,
    );
  }

  async frontendProposalDetail(
    namespace: string,
    slug: string,
    proposalId: number,
  ): Promise<Envelope<ProposalDetailReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${this.path(namespace, slug)}/proposals/${this.path(proposalId)}`,
    );
  }

  // ── Review/Promotion mutation methods ────────────────────────────

  /** Approve a pending review task. */
  async approveReview(
    id: number,
    req?: ReviewMutationRequest,
  ): Promise<Envelope<ReviewMutationResponse>> {
    return this.fetch(`/api/v1/reviews/${this.path(id)}/approve`, {
      method: "POST",
      body: req ? JSON.stringify(req) : "{}",
    });
  }

  /** Reject a pending review task. */
  async rejectReview(
    id: number,
    req?: ReviewMutationRequest,
  ): Promise<Envelope<ReviewMutationResponse>> {
    return this.fetch(`/api/v1/reviews/${this.path(id)}/reject`, {
      method: "POST",
      body: req ? JSON.stringify(req) : "{}",
    });
  }

  /** Withdraw a pending review task. */
  async withdrawReview(
    id: number,
  ): Promise<Envelope<WithdrawResponse>> {
    return this.fetch(`/api/v1/reviews/${this.path(id)}/withdraw`, {
      method: "POST",
      body: "{}",
    });
  }

  /** Approve a pending promotion request. */
  async approvePromotion(
    id: number,
    req?: PromotionMutationRequest,
  ): Promise<Envelope<PromotionMutationResponse>> {
    return this.fetch(`/api/v1/promotions/${this.path(id)}/approve`, {
      method: "POST",
      body: req ? JSON.stringify(req) : "{}",
    });
  }

  /** Reject a pending promotion request. */
  async rejectPromotion(
    id: number,
    req?: PromotionMutationRequest,
  ): Promise<Envelope<PromotionMutationResponse>> {
    return this.fetch(`/api/v1/promotions/${this.path(id)}/reject`, {
      method: "POST",
      body: req ? JSON.stringify(req) : "{}",
    });
  }

  /** Withdraw a pending promotion request. */
  async withdrawPromotion(
    id: number,
  ): Promise<Envelope<WithdrawResponse>> {
    return this.fetch(`/api/v1/promotions/${this.path(id)}/withdraw`, {
      method: "POST",
      body: "{}",
    });
  }

  // ── Pagination iterators ──────────────────────────────────────────

  /** Bounded async iterator for frontend search results. */
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

  /** Bounded async iterator for review queue. */
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

  /** Bounded async iterator for promotion queue. */
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

  /** Bounded async iterator for issue list. */
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

  /** Bounded async iterator for discussion list. */
  async *iterFrontendDiscussions(
    namespace: string,
    slug: string,
    options?: PageIteratorOptions,
  ): AsyncGenerator<DiscussionListReadModel> {
    let page = options?.page ?? 0;
    const size = options?.size ?? 20;
    const maxPages = options?.maxPages ?? 10;
    for (let i = 0; i < maxPages; i++) {
      const result = await this.frontendDiscussionList(
        namespace,
        slug,
        page,
        size,
      );
      if (!result.success || !result.data) return;
      yield result.data;
      if (result.data.discussions.length === 0) return;
      if (result.data.totalCount <= (page + 1) * size) return;
      page++;
    }
  }

  /** Bounded async iterator for proposal list. */
  async *iterFrontendProposals(
    namespace: string,
    slug: string,
    options?: PageIteratorOptions,
  ): AsyncGenerator<ProposalListReadModel> {
    let page = options?.page ?? 0;
    const size = options?.size ?? 20;
    const maxPages = options?.maxPages ?? 10;
    for (let i = 0; i < maxPages; i++) {
      const result = await this.frontendProposalList(
        namespace,
        slug,
        page,
        size,
      );
      if (!result.success || !result.data) return;
      yield result.data;
      if (result.data.proposals.length === 0) return;
      if (result.data.totalCount <= (page + 1) * size) return;
      page++;
    }
  }

  /** Bounded async iterator for release list. */
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

  /** Bounded async iterator for pipeline runs. */
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
