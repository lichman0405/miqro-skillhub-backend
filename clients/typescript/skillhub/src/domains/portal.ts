import type { SkillHubTransport, Envelope } from "../core.js";
import type { SearchQuery, SearchResult, SkillDetail } from "../types/common.js";

export function search(
  transport: SkillHubTransport,
  query: SearchQuery,
): Promise<Envelope<SearchResult>> {
  return transport.request(
    `/api/v1/search${transport.query({
      keyword: query.keyword,
      sortBy: query.sortBy,
      page: query.page,
      size: query.size,
      labelSlugs: query.labelSlugs,
      installableOnly: query.installableOnly,
    })}`,
  );
}

export function getSkill(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
): Promise<Envelope<SkillDetail>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}`,
  );
}

export function getNamespace(
  transport: SkillHubTransport,
  slug: string,
): Promise<Envelope<unknown>> {
  return transport.request(`/api/v1/namespaces/${transport.path(slug)}`);
}
