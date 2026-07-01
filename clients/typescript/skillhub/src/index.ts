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
}

export default SkillHubClient;
