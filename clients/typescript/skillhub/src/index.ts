/**
 * SkillHub TypeScript client — generated from the OpenAPI contract.
 *
 * This client mirrors the Go HTTP route surfaces and provides
 * strongly-typed request/response shapes for frontend consumers.
 */

/** Standard response envelope shared by every SkillHub HTTP endpoint. */
export interface Envelope<T = unknown> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
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
  namespaceId: number;
  submittedBy: string;
  status: string;
  submittedAt: string;
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
  sourceVersionId: number;
  targetNamespaceId: number;
  submittedBy: string;
  status: string;
  submittedAt: string;
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

// ── Client ─────────────────────────────────────────────────────────────

/** SkillHub API client — thin HTTP wrapper over the backend. */
export class SkillHubClient {
  constructor(private baseUrl: string = "http://localhost:8080") {}

  private async fetch<T>(
    path: string,
    init?: RequestInit
  ): Promise<Envelope<T>> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      ...init,
      headers: { "Content-Type": "application/json", ...init?.headers },
    });
    return res.json() as Promise<Envelope<T>>;
  }

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

  /** Search skills. */
  async search(query: SearchQuery): Promise<Envelope<SearchResult>> {
    const params = new URLSearchParams();
    if (query.keyword) params.set("keyword", query.keyword);
    if (query.sortBy) params.set("sortBy", query.sortBy);
    if (query.installableOnly) params.set("installableOnly", "true");
    return this.fetch(`/api/v1/search?${params.toString()}`);
  }

  /** Get a skill by namespace and slug. */
  async getSkill(namespace: string, slug: string) {
    return this.fetch<SkillDetail>(`/api/v1/skills/${namespace}/${slug}`);
  }

  /** Get a namespace by slug. */
  async getNamespace(slug: string) {
    return this.fetch(`/api/v1/namespaces/${slug}`);
  }

  // ── Frontend page methods ────────────────────────────────────────────

  /** Get registry search/home page read model. */
  async frontendSearch(): Promise<Envelope<RegistrySearchReadModel>> {
    return this.fetch("/api/v1/frontend/search");
  }

  /** Get skill detail page read model (viewer-scoped). */
  async frontendSkillDetail(
    namespace: string,
    slug: string
  ): Promise<Envelope<SkillDetailReadModel>> {
    return this.fetch(`/api/v1/frontend/skills/${namespace}/${slug}`);
  }

  /** Get version detail/compare page read model. */
  async frontendVersionDetail(
    namespace: string,
    slug: string,
    version: string
  ): Promise<Envelope<VersionDetailReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${namespace}/${slug}/versions/${version}`
    );
  }

  /** Get publish validate page read model. */
  async frontendPublishValidate(
    namespace: string
  ): Promise<Envelope<PublishValidateReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${namespace}/publish/validate`
    );
  }

  /** Get namespace list page read model. */
  async frontendNamespaces(): Promise<Envelope<NamespaceListReadModel>> {
    return this.fetch("/api/v1/frontend/namespaces");
  }

  /** Get namespace detail page read model (viewer-scoped). */
  async frontendNamespaceDetail(
    slug: string
  ): Promise<Envelope<NamespaceDetailReadModel>> {
    return this.fetch(`/api/v1/frontend/namespaces/${slug}`);
  }

  /** Get review queue page read model. */
  async frontendReviews(): Promise<Envelope<ReviewQueueReadModel>> {
    return this.fetch("/api/v1/frontend/reviews");
  }

  /** Get review detail page read model. */
  async frontendReviewDetail(
    id: number
  ): Promise<Envelope<ReviewDetailReadModel>> {
    return this.fetch(`/api/v1/frontend/reviews/${id}`);
  }

  /** Get promotion queue page read model. */
  async frontendPromotions(): Promise<Envelope<PromotionQueueReadModel>> {
    return this.fetch("/api/v1/frontend/promotions");
  }

  /** Get promotion detail page read model. */
  async frontendPromotionDetail(
    id: number
  ): Promise<Envelope<PromotionDetailReadModel>> {
    return this.fetch(`/api/v1/frontend/promotions/${id}`);
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
    entries: PackageEntry[]
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
    version?: string
  ): Promise<Envelope<ResolveResult>> {
    const params = version ? `?version=${encodeURIComponent(version)}` : "";
    return this.fetch(
      `/api/tool/v1/skills/${namespace}/${slug}/resolve${params}`
    );
  }

  /** Get install target metadata (miqro install). */
  async toolInstall(
    namespace: string,
    slug: string,
    version?: string
  ): Promise<Envelope<InstallTarget>> {
    const params = version ? `?version=${encodeURIComponent(version)}` : "";
    return this.fetch(
      `/api/tool/v1/skills/${namespace}/${slug}/install${params}`
    );
  }

  /** Diff two skill versions (miqro diff). */
  async toolDiff(
    namespace: string,
    slug: string,
    fromVersion: string,
    toVersion: string
  ): Promise<Envelope<VersionDiff>> {
    const params = `?from=${encodeURIComponent(fromVersion)}&to=${encodeURIComponent(toVersion)}`;
    return this.fetch(
      `/api/tool/v1/skills/${namespace}/${slug}/diff${params}`
    );
  }

  /** Validate a skill package (miqro validate). Accepts a zip File/Blob. */
  async toolValidate(
    namespace: string,
    zipFile: Blob
  ): Promise<Envelope<ToolValidateResponse>> {
    const formData = new FormData();
    formData.append("package", zipFile);
    return this.fetch(`/api/tool/v1/skills/${namespace}/validate`, {
      method: "POST",
      body: formData,
      headers: {}, // let browser set multipart Content-Type
    });
  }

  /** Publish a skill package (miqro publish). Accepts a zip File/Blob. */
  async toolPublish(
    namespace: string,
    zipFile: Blob
  ): Promise<Envelope<ToolPublishResponse>> {
    const formData = new FormData();
    formData.append("package", zipFile);
    return this.fetch(`/api/tool/v1/skills/${namespace}/publish`, {
      method: "POST",
      body: formData,
      headers: {}, // let browser set multipart Content-Type
    });
  }

  /** Trigger skill evaluation (miqro evaluate — Phase 12 placeholder). */
  async toolEvaluate(
    req: EvaluateRequest
  ): Promise<Envelope<EvaluateResponse>> {
    return this.fetch("/api/tool/v1/evaluate/trigger", {
      method: "POST",
      body: JSON.stringify(req),
    });
  }

  /** Prepare a skill change proposal (miqro propose — Phase 11 placeholder). */
  async toolPropose(
    req: ProposalRequest
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
    size?: number
  ): Promise<Envelope<ReleaseListResult>> {
    const params = new URLSearchParams();
    if (page !== undefined) params.set("page", String(page));
    if (size !== undefined) params.set("size", String(size));
    return this.fetch(
      `/api/v1/skills/${namespace}/${slug}/releases?${params.toString()}`
    );
  }

  /** Get latest stable release for a skill. */
  async getLatestRelease(
    namespace: string,
    slug: string,
    channel?: string
  ): Promise<Envelope<Release>> {
    const params = new URLSearchParams();
    if (channel) params.set("channel", channel);
    return this.fetch(
      `/api/v1/skills/${namespace}/${slug}/releases/latest?${params.toString()}`
    );
  }

  /** Get a single release by ID. */
  async getRelease(
    namespace: string,
    slug: string,
    releaseId: number
  ): Promise<Envelope<ReleaseDetailResponse>> {
    return this.fetch(
      `/api/v1/skills/${namespace}/${slug}/releases/${releaseId}`
    );
  }

  /** Create a new release. */
  async createRelease(
    namespace: string,
    slug: string,
    req: CreateReleaseRequest
  ): Promise<Envelope<Release>> {
    return this.fetch(`/api/v1/skills/${namespace}/${slug}/releases`, {
      method: "POST",
      body: JSON.stringify(req),
    });
  }

  /** Update release metadata. */
  async updateRelease(
    namespace: string,
    slug: string,
    releaseId: number,
    req: UpdateReleaseRequest
  ): Promise<Envelope<Release>> {
    return this.fetch(
      `/api/v1/skills/${namespace}/${slug}/releases/${releaseId}`,
      { method: "PATCH", body: JSON.stringify(req) }
    );
  }

  /** Delete a release. */
  async deleteRelease(
    namespace: string,
    slug: string,
    releaseId: number
  ): Promise<Envelope<{ status: string }>> {
    return this.fetch(
      `/api/v1/skills/${namespace}/${slug}/releases/${releaseId}`,
      { method: "DELETE" }
    );
  }

  // ── Frontend release page methods ──────────────────────────────────────

  /** Get release list page read model. */
  async frontendReleaseList(
    namespace: string,
    slug: string
  ): Promise<Envelope<ReleaseListReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${namespace}/${slug}/releases`
    );
  }

  /** Get release detail page read model. */
  async frontendReleaseDetail(
    namespace: string,
    slug: string,
    releaseId: number
  ): Promise<Envelope<ReleaseDetailReadModel>> {
    return this.fetch(
      `/api/v1/frontend/skills/${namespace}/${slug}/releases/${releaseId}`
    );
  }
}

export default SkillHubClient;
