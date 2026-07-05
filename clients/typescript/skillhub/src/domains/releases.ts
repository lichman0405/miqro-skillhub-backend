import type { SkillHubTransport, Envelope } from "../core.js";
import type {
  Release,
  ReleaseListResult,
  ReleaseDetailResponse,
  CreateReleaseRequest,
  UpdateReleaseRequest,
} from "../types/releases.js";

export function listReleases(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  page?: number,
  size?: number,
): Promise<Envelope<ReleaseListResult>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/releases${transport.query({ page, size })}`,
  );
}

export function getLatestRelease(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  channel?: string,
): Promise<Envelope<Release>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/releases/latest${transport.query({ channel })}`,
  );
}

export function getRelease(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  releaseId: number,
): Promise<Envelope<ReleaseDetailResponse>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/releases/${transport.path(releaseId)}`,
  );
}

export function createRelease(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  req: CreateReleaseRequest,
): Promise<Envelope<Release>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/releases`,
    { method: "POST", body: JSON.stringify(req) },
  );
}

export function updateRelease(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  releaseId: number,
  req: UpdateReleaseRequest,
): Promise<Envelope<Release>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/releases/${transport.path(releaseId)}`,
    { method: "PATCH", body: JSON.stringify(req) },
  );
}

export function deleteRelease(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  releaseId: number,
): Promise<Envelope<{ status: string }>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/releases/${transport.path(releaseId)}`,
    { method: "DELETE" },
  );
}

export function publishRelease(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  releaseId: number,
): Promise<Envelope<Release>> {
  return transport.request(
    `/api/v1/skills/${transport.path(namespace, slug)}/releases/${transport.path(releaseId)}/publish`,
    { method: "POST" },
  );
}
