import type { SkillHubTransport, Envelope } from "../core.js";
import type {
  PipelineRun,
  PipelineRunListResult,
  CheckRun,
  CheckArtifact,
  GateEvalResult,
} from "../types/agentci.js";

export function listPipelineRuns(
  transport: SkillHubTransport,
  skillId: number,
  page?: number,
  size?: number,
): Promise<Envelope<PipelineRunListResult>> {
  return transport.request(
    `/api/v1/skills/${transport.path(skillId)}/ci/runs${transport.query({ page, size })}`,
  );
}

export function getPipelineRun(
  transport: SkillHubTransport,
  skillId: number,
  runId: number,
): Promise<Envelope<PipelineRun>> {
  return transport.request(
    `/api/v1/skills/${transport.path(skillId)}/ci/runs/${transport.path(runId)}`,
  );
}

export function listCheckRuns(
  transport: SkillHubTransport,
  skillId: number,
  runId: number,
): Promise<Envelope<CheckRun[]>> {
  return transport.request(
    `/api/v1/skills/${transport.path(skillId)}/ci/runs/${transport.path(runId)}/checks`,
  );
}

export function getCheckRun(
  transport: SkillHubTransport,
  skillId: number,
  checkId: number,
): Promise<Envelope<CheckRun>> {
  return transport.request(
    `/api/v1/skills/${transport.path(skillId)}/ci/checks/${transport.path(checkId)}`,
  );
}

export function listCheckArtifacts(
  transport: SkillHubTransport,
  skillId: number,
  checkId: number,
): Promise<Envelope<CheckArtifact[]>> {
  return transport.request(
    `/api/v1/skills/${transport.path(skillId)}/ci/checks/${transport.path(checkId)}/artifacts`,
  );
}

export function evaluateGates(
  transport: SkillHubTransport,
  skillId: number,
  params?: { trigger?: string; versionId?: number; releaseId?: number },
): Promise<Envelope<GateEvalResult>> {
  return transport.request(
    `/api/v1/skills/${transport.path(skillId)}/ci/gates${transport.query({
      trigger: params?.trigger,
      versionId: params?.versionId,
      releaseId: params?.releaseId,
    })}`,
  );
}
