import type { SkillHubTransport, Envelope } from "../core.js";
import type {
  CreateIssueRequest,
  UpdateIssueRequest,
  CommunitySearchResult,
} from "../types/community.js";

export function listIssues(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  params?: { status?: string; page?: number; size?: number },
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/issues${transport.query({
      status: params?.status,
      page: params?.page,
      size: params?.size,
    })}`,
  );
}

export function getIssue(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  issueId: number,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/issues/${transport.path(issueId)}`,
  );
}

export function createIssue(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  body: CreateIssueRequest,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/issues`,
    { method: "POST", body: JSON.stringify(body) },
  );
}

export function updateIssue(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  issueId: number,
  body: UpdateIssueRequest,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/issues/${transport.path(issueId)}`,
    { method: "PATCH", body: JSON.stringify(body) },
  );
}

export function deleteIssue(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  issueId: number,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/issues/${transport.path(issueId)}`,
    { method: "DELETE" },
  );
}

export function listIssueComments(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  issueId: number,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/issues/${transport.path(issueId)}/comments`,
  );
}

export function addIssueComment(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  issueId: number,
  body: { body: string },
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/issues/${transport.path(issueId)}/comments`,
    { method: "POST", body: JSON.stringify(body) },
  );
}

export function listDiscussions(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  params?: { category?: string; page?: number; size?: number },
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/discussions${transport.query({
      category: params?.category,
      page: params?.page,
      size: params?.size,
    })}`,
  );
}

export function getDiscussion(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  discussionId: number,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/discussions/${transport.path(discussionId)}`,
  );
}

export function createDiscussion(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  body: { title: string; body?: string; category?: string },
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/discussions`,
    { method: "POST", body: JSON.stringify(body) },
  );
}

export function updateDiscussion(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  discussionId: number,
  body: Record<string, unknown>,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/discussions/${transport.path(discussionId)}`,
    { method: "PATCH", body: JSON.stringify(body) },
  );
}

export function deleteDiscussion(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  discussionId: number,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/discussions/${transport.path(discussionId)}`,
    { method: "DELETE" },
  );
}

export function listDiscussionComments(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  discussionId: number,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/discussions/${transport.path(discussionId)}/comments`,
  );
}

export function addDiscussionComment(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  discussionId: number,
  body: { body: string },
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/discussions/${transport.path(discussionId)}/comments`,
    { method: "POST", body: JSON.stringify(body) },
  );
}

export function acceptAnswer(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  discussionId: number,
  commentId: number,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/discussions/${transport.path(discussionId)}/accept-answer`,
    { method: "POST", body: JSON.stringify({ commentId }) },
  );
}

export function listWikiPages(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/wiki`,
  );
}

export function getWikiPage(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  pageSlug: string,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/wiki/${transport.path(pageSlug)}`,
  );
}

export function createWikiPage(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  body: { title: string; slug: string; body: string; changeSummary?: string },
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/wiki`,
    { method: "POST", body: JSON.stringify(body) },
  );
}

export function updateWikiPage(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  pageSlug: string,
  body: { body: string; changeSummary?: string },
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/wiki/${transport.path(pageSlug)}`,
    { method: "PUT", body: JSON.stringify(body) },
  );
}

export function listWikiVersions(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  pageSlug: string,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/wiki/${transport.path(pageSlug)}/versions`,
  );
}

export function listProposals(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  params?: { status?: string; page?: number; size?: number },
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/proposals${transport.query({
      status: params?.status,
      page: params?.page,
      size: params?.size,
    })}`,
  );
}

export function getProposal(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  proposalId: number,
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/proposals/${transport.path(proposalId)}`,
  );
}

export function createProposal(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  body: { title: string; summary?: string; proposedChangesJSON?: string },
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/proposals`,
    { method: "POST", body: JSON.stringify(body) },
  );
}

export function updateProposal(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  proposalId: number,
  body: { status?: string; comment?: string },
): Promise<Envelope<unknown>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/proposals/${transport.path(proposalId)}`,
    { method: "PATCH", body: JSON.stringify(body) },
  );
}

export function communitySearch(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  params?: { query?: string; types?: string; page?: number; size?: number },
): Promise<Envelope<CommunitySearchResult>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/community/search${transport.query({
      query: params?.query,
      types: params?.types,
      page: params?.page,
      size: params?.size,
    })}`,
  );
}
