/**
 * SkillHub TypeScript client — public barrel entry point.
 *
 * Internally modularized under src/domains/ and src/types/. Consumers should
 * import from "@miqro/skillhub-client"; internal module paths are not part
 * of the public compatibility contract.
 */

// ── Core types and classes ──────────────────────────────────────────────

import { SkillHubClient } from "./client.js";
export { SkillHubClient };
export {
  SkillHubError,
  type Envelope,
  type SkillHubClientConfig,
  type SkillHubClientOptions,
} from "./core.js";

// ── Pagination ──────────────────────────────────────────────────────────

export type { PageIteratorOptions } from "./pagination.js";

// ── Shared domain types ────────────────────────────────────────────────

export type {
  Principal,
  SearchQuery,
  SearchResult,
  SkillDetail,
  VersionDetail,
  SkillFile,
  Namespace,
  NamespaceMember,
} from "./types/common.js";

// ── Frontend read-model types ───────────────────────────────────────────

export type {
  RegistrySearchActions,
  RegistrySearchReadModel,
  SkillDetailActions,
  SkillDetailReadModel,
  VersionActions,
  VersionDetailReadModel,
  PublishValidateActions,
  PublishValidateReadModel,
  NamespaceListActions,
  NamespaceListReadModel,
  NamespaceDetailActions,
  NamespaceDetailReadModel,
  ReviewTaskView,
  ReviewQueueActions,
  ReviewQueueReadModel,
  ReviewDetailActions,
  ReviewDetailReadModel,
  PromotionRequestView,
  PromotionQueueActions,
  PromotionQueueReadModel,
  PromotionDetailActions,
  PromotionDetailReadModel,
  GovernanceSummaryView,
  GovernanceActivityView,
  GovernanceWorkbenchActions,
  GovernanceWorkbenchReadModel,
  AdminStatsView,
  AdminPageActions,
  AdminPageReadModel,
} from "./types/frontend.js";

// ── Tool API types ─────────────────────────────────────────────────────

export type {
  PackageEntry,
  ManifestEntry,
  PackageManifest,
  PackageHashRequest,
  PackageHashResponse,
  WorkspaceMetadataResponse,
  ResolveResult,
  AgentRuntime,
  InstallTarget,
  DiffSummary,
  DiffLine,
  DiffHunk,
  DiffFile,
  VersionDiff,
  ToolValidateResponse,
  ToolPublishResponse,
  EvaluateRequest,
  EvaluateResponse,
  ProposalRequest,
  ProposalResponse,
} from "./types/tooling.js";

// ── Release types ──────────────────────────────────────────────────────

export type {
  Release,
  ReleaseAsset,
  ReleaseListResult,
  ReleaseDetailResponse,
  CreateReleaseRequest,
  UpdateReleaseRequest,
  ReleaseListReadModel,
  ReleaseListView,
  ReleaseListActions,
  ReleaseDetailReadModel,
  ReleaseDetailView,
  ReleaseAssetView,
  ReleaseDetailActions,
} from "./types/releases.js";

// ── Community types ────────────────────────────────────────────────────

export type {
  Issue,
  IssueComment,
  CreateIssueRequest,
  UpdateIssueRequest,
  Discussion,
  DiscussionComment,
  WikiPage,
  WikiPageVersion,
  ChangeProposal,
  IssueListView,
  IssueListActions,
  IssueListReadModel,
  IssueDetailView,
  IssueDetailActions,
  IssueDetailReadModel,
  CommentView,
  DiscussionListView,
  DiscussionListActions,
  DiscussionListReadModel,
  DiscussionDetailView,
  DiscussionDetailActions,
  DiscussionDetailReadModel,
  WikiPageListView,
  WikiPageListActions,
  WikiPageListReadModel,
  WikiPageDetailView,
  WikiVersionView,
  WikiPageDetailActions,
  WikiPageDetailReadModel,
  ProposalListView,
  ProposalListActions,
  ProposalListReadModel,
  ProposalDetailView,
  ProposalDetailActions,
  ProposalDetailReadModel,
  CommunitySearchResultItem,
  CommunitySearchResult,
} from "./types/community.js";

// ── Agent CI types ─────────────────────────────────────────────────────

export type {
  PipelineRun,
  PipelineRunListResult,
  CheckRun,
  CheckArtifact,
  GatePolicyResult,
  GateEvalResult,
} from "./types/agentci.js";

// ── Mutation types ────────────────────────────────────────────────────

export type {
  ReviewMutationRequest,
  ReviewMutationResponse,
  PromotionMutationRequest,
  PromotionMutationResponse,
  WithdrawResponse,
} from "./types/mutations.js";

export default SkillHubClient;
