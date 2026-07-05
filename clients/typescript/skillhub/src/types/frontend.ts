import type { SearchResult, SkillDetail, VersionDetail, SkillFile, Namespace, NamespaceMember } from "./common.js";

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

/** Available actions for namespace listing. */
export interface NamespaceListActions {
  canCreateNamespace: boolean;
}

/** Namespace list read model. */
export interface NamespaceListReadModel {
  namespaces: Namespace[];
  availableActions: NamespaceListActions;
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
