import type { SkillHubTransport, Envelope } from "../core.js";
import type {
  PackageEntry,
  PackageHashResponse,
  WorkspaceMetadataResponse,
  ResolveResult,
  InstallTarget,
  VersionDiff,
  ToolValidateResponse,
  ToolPublishResponse,
  EvaluateRequest,
  EvaluateResponse,
  ProposalRequest,
  ProposalResponse,
} from "../types/tooling.js";

export function toolWorkspaceMetadata(
  transport: SkillHubTransport,
): Promise<Envelope<WorkspaceMetadataResponse>> {
  return transport.request("/api/tool/v1/workspace/metadata");
}

export function toolPackageHash(
  transport: SkillHubTransport,
  entries: PackageEntry[],
): Promise<Envelope<PackageHashResponse>> {
  return transport.request("/api/tool/v1/packages/hash", {
    method: "POST",
    body: JSON.stringify({ entries }),
  });
}

export function toolResolve(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  version?: string,
): Promise<Envelope<ResolveResult>> {
  return transport.request(
    `/api/tool/v1/skills/${transport.path(namespace, slug)}/resolve${transport.query({ version })}`,
  );
}

export function toolInstall(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  version?: string,
): Promise<Envelope<InstallTarget>> {
  return transport.request(
    `/api/tool/v1/skills/${transport.path(namespace, slug)}/install${transport.query({ version })}`,
  );
}

export function toolDiff(
  transport: SkillHubTransport,
  namespace: string,
  slug: string,
  fromVersion: string,
  toVersion: string,
): Promise<Envelope<VersionDiff>> {
  return transport.request(
    `/api/tool/v1/skills/${transport.path(namespace, slug)}/diff${transport.query({ from: fromVersion, to: toVersion })}`,
  );
}

export function toolValidate(
  transport: SkillHubTransport,
  namespace: string,
  zipFile: Blob,
): Promise<Envelope<ToolValidateResponse>> {
  const formData = new FormData();
  formData.append("package", zipFile);
  return transport.request(
    `/api/tool/v1/skills/${transport.path(namespace)}/validate`,
    { method: "POST", body: formData, headers: {} },
  );
}

export function toolPublish(
  transport: SkillHubTransport,
  namespace: string,
  zipFile: Blob,
): Promise<Envelope<ToolPublishResponse>> {
  const formData = new FormData();
  formData.append("package", zipFile);
  return transport.request(
    `/api/tool/v1/skills/${transport.path(namespace)}/publish`,
    { method: "POST", body: formData, headers: {} },
  );
}

export function toolEvaluate(
  transport: SkillHubTransport,
  req: EvaluateRequest,
): Promise<Envelope<EvaluateResponse>> {
  return transport.request("/api/tool/v1/evaluate/trigger", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export function toolPropose(
  transport: SkillHubTransport,
  req: ProposalRequest,
): Promise<Envelope<ProposalResponse>> {
  return transport.request("/api/tool/v1/proposals/prepare", {
    method: "POST",
    body: JSON.stringify(req),
  });
}
