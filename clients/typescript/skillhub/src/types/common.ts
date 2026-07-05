/** Core domain types used across multiple SDK surfaces. */

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

/** A namespace. */
export interface Namespace {
  id: number;
  slug: string;
  displayName: string;
  type: string;
  description: string;
}

/** A namespace member. */
export interface NamespaceMember {
  namespaceId: number;
  userId: string;
  role: string;
}
