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
}

export default SkillHubClient;
