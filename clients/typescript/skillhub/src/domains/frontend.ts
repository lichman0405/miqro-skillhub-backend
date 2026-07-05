import type { SkillHubTransport, Envelope } from "../core.js";
import type { SearchQuery } from "../types/common.js";
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
} from "../types/frontend.js";
import type {
  IssueListReadModel,
  IssueDetailReadModel,
  DiscussionListReadModel,
  DiscussionDetailReadModel,
  WikiPageListReadModel,
  WikiPageDetailReadModel,
  ProposalListReadModel,
  ProposalDetailReadModel,
} from "../types/community.js";
import type { ReleaseListReadModel, ReleaseDetailReadModel } from "../types/releases.js";

// ── Core frontend page methods ─────────────────────────────────────────

export function frontendSearch(
  transport: SkillHubTransport,
  query: SearchQuery = {},
): Promise<Envelope<RegistrySearchReadModel>> {
  return transport.request(
    `/api/v1/frontend/search${transport.query({
      q: query.keyword,
      sort: query.sortBy,
      page: query.page,
      size: query.size,
      labels: query.labelSlugs,
      installable: query.installableOnly,
    })}`,
  );
}

export function frontendSkillDetail(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
): Promise<Envelope<SkillDetailReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}`,
  );
}

export function frontendVersionDetail(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  version: string,
): Promise<Envelope<VersionDetailReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}/versions/${transport.path(version)}`,
  );
}

export function frontendPublishValidate(
  transport: SkillHubTransport,
  namespace: string,
): Promise<Envelope<PublishValidateReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace)}/publish/validate`,
  );
}

export function frontendNamespaces(
  transport: SkillHubTransport,
): Promise<Envelope<NamespaceListReadModel>> {
  return transport.request("/api/v1/frontend/namespaces");
}

export function frontendNamespaceDetail(
  transport: SkillHubTransport,
  slug: string,
): Promise<Envelope<NamespaceDetailReadModel>> {
  return transport.request(`/api/v1/frontend/namespaces/${transport.path(slug)}`);
}

// ── Review/promotion queue pages ───────────────────────────────────────

export function frontendReviews(
  transport: SkillHubTransport,
  page?: number,
  size?: number,
): Promise<Envelope<ReviewQueueReadModel>> {
  return transport.request(
    `/api/v1/frontend/reviews${transport.query({ page, size })}`,
  );
}

export function frontendReviewDetail(
  transport: SkillHubTransport,
  id: number,
): Promise<Envelope<ReviewDetailReadModel>> {
  return transport.request(`/api/v1/frontend/reviews/${transport.path(id)}`);
}

export function frontendPromotions(
  transport: SkillHubTransport,
  page?: number,
  size?: number,
): Promise<Envelope<PromotionQueueReadModel>> {
  return transport.request(
    `/api/v1/frontend/promotions${transport.query({ page, size })}`,
  );
}

export function frontendPromotionDetail(
  transport: SkillHubTransport,
  id: number,
): Promise<Envelope<PromotionDetailReadModel>> {
  return transport.request(`/api/v1/frontend/promotions/${transport.path(id)}`);
}

// ── Governance / Admin ─────────────────────────────────────────────────

export function frontendGovernance(
  transport: SkillHubTransport,
): Promise<Envelope<GovernanceWorkbenchReadModel>> {
  return transport.request("/api/v1/frontend/governance");
}

export function frontendAdmin(
  transport: SkillHubTransport,
): Promise<Envelope<AdminPageReadModel>> {
  return transport.request("/api/v1/frontend/admin");
}

// ── Frontend release pages ─────────────────────────────────────────────

export function frontendReleaseList(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
): Promise<Envelope<ReleaseListReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}/releases`,
  );
}

export function frontendReleaseDetail(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  releaseId: number,
): Promise<Envelope<ReleaseDetailReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}/releases/${transport.path(releaseId)}`,
  );
}

// ── Community frontend pages ───────────────────────────────────────────

export function frontendIssueList(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  page?: number,
  size?: number,
): Promise<Envelope<IssueListReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}/issues${transport.query({ page, size })}`,
  );
}

export function frontendIssueDetail(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  issueId: number,
): Promise<Envelope<IssueDetailReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}/issues/${transport.path(issueId)}`,
  );
}

export function frontendDiscussionList(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  page?: number,
  size?: number,
): Promise<Envelope<DiscussionListReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}/discussions${transport.query({ page, size })}`,
  );
}

export function frontendDiscussionDetail(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  discussionId: number,
): Promise<Envelope<DiscussionDetailReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}/discussions/${transport.path(discussionId)}`,
  );
}

export function frontendWikiList(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
): Promise<Envelope<WikiPageListReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}/wiki`,
  );
}

export function frontendWikiDetail(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  pageSlug: string,
): Promise<Envelope<WikiPageDetailReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}/wiki/${transport.path(pageSlug)}`,
  );
}

export function frontendProposalList(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  page?: number,
  size?: number,
): Promise<Envelope<ProposalListReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}/proposals${transport.query({ page, size })}`,
  );
}

export function frontendProposalDetail(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  proposalId: number,
): Promise<Envelope<ProposalDetailReadModel>> {
  return transport.request(
    `/api/v1/frontend/skills/${transport.path(namespace, slug)}/proposals/${transport.path(proposalId)}`,
  );
}
