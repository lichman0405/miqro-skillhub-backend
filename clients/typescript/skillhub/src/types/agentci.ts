/** A single CI pipeline run. */
export interface PipelineRun {
  id: number;
  pipelineId: number;
  skillId: number;
  versionId?: number;
  releaseId?: number;
  triggerType: string;
  triggeredBy: string;
  status: string;
  checkCount: number;
  passedCount: number;
  failedCount: number;
  skippedCount: number;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
}

/** Paginated pipeline run list. */
export interface PipelineRunListResult {
  runs: PipelineRun[];
  totalCount: number;
  page: number;
  size: number;
}

/** A single CI check run. */
export interface CheckRun {
  id: number;
  pipelineRunId: number;
  skillId: number;
  versionId?: number;
  releaseId?: number;
  name: string;
  runnerType: string;
  status: string;
  conclusion?: string;
  summary?: string;
  isBlocking: boolean;
  startedAt?: string;
  completedAt?: string;
  durationMs?: number;
  createdAt: string;
  updatedAt: string;
}

/** A CI check artifact. */
export interface CheckArtifact {
  id: number;
  checkRunId: number;
  name: string;
  contentType: string;
  size: number;
  storageKey: string;
  createdAt: string;
}

/** A single gate policy evaluation result. */
export interface GatePolicyResult {
  policyId: number;
  policyName: string;
  passed: boolean;
  reason?: string;
}

/** Gate evaluation result. */
export interface GateEvalResult {
  passed: boolean;
  reason?: string;
  policyResults?: GatePolicyResult[];
}
