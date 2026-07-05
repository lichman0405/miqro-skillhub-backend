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

export interface CommunitySearchResultItem {
  type: string; id: number; skillId: number; title: string; snippet?: string;
}
export interface CommunitySearchResult {
  items: CommunitySearchResultItem[]; totalCount: number; page: number; size: number;
}
