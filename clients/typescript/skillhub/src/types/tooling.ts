/** Package entry for manifest/hash computation. */
export interface PackageEntry {
  path: string;
  content: string;
  size: number;
  contentType: string;
}

/** A single entry in a deterministic package manifest. */
export interface ManifestEntry {
  path: string;
  size: number;
  contentType: string;
  sha256: string;
}

/** Deterministic package manifest. */
export interface PackageManifest {
  entries: ManifestEntry[];
  hash: string;
  totalSize: number;
  fileCount: number;
}

/** Request to compute deterministic package hash. */
export interface PackageHashRequest {
  entries: PackageEntry[];
}

/** Response from package hash computation. */
export interface PackageHashResponse {
  manifest: PackageManifest;
}

/** Workspace metadata response (miqro init contract). */
export interface WorkspaceMetadataResponse {
  workspace: {
    requiredFiles: string[];
    optionalFiles: string[];
    manifestFormat: string;
    schema: {
      fields: string[];
      required: string[];
    };
  };
}

/** Resolved version with tooling fingerprint. */
export interface ResolveResult {
  skillId: number;
  namespace: string;
  slug: string;
  version: string;
  versionId: number;
  fingerprint: string;
  downloadUrl: string;
}

/** Agent runtime descriptor. */
export interface AgentRuntime {
  type: string;
  minVersion?: string;
}

/** Install target metadata. */
export interface InstallTarget {
  skillId: number;
  skillSlug: string;
  namespace: string;
  version: string;
  fingerprint: string;
  downloadUrl: string;
  supportedAgents?: AgentRuntime[];
  installPath?: string;
}

/** Diff summary counts. */
export interface DiffSummary {
  totalFiles: number;
  addedFiles: number;
  modifiedFiles: number;
  removedFiles: number;
  addedLines: number;
  removedLines: number;
}

/** A single line in a diff hunk. */
export interface DiffLine {
  type: "ADD" | "DELETE" | "CONTEXT";
  content: string;
  oldLineNumber?: number;
  newLineNumber?: number;
}

/** A contiguous change block. */
export interface DiffHunk {
  oldStart: number;
  oldLines: number;
  newStart: number;
  newLines: number;
  lines: DiffLine[];
}

/** A single file in a version diff. */
export interface DiffFile {
  path: string;
  changeType: "ADDED" | "REMOVED" | "MODIFIED";
  oldSize?: number;
  newSize?: number;
  binary: boolean;
  truncated: boolean;
  hunks?: DiffHunk[];
}

/** Full version diff. */
export interface VersionDiff {
  fromVersion: string;
  toVersion: string;
  summary: DiffSummary;
  files: DiffFile[];
}

/** Tool-facing validation result. */
export interface ToolValidateResponse {
  valid: boolean;
  errors?: string[];
  warnings?: string[];
  resolvedSlug?: string;
  resolvedVersion?: string;
}

/** Tool-facing publish response. */
export interface ToolPublishResponse {
  skillId: number;
  slug: string;
  version: {
    id: number;
    version: string;
    status: string;
  };
}

/** Evaluate trigger request (placeholder). */
export interface EvaluateRequest {
  skillId: number;
  versionId: number;
  trigger: string;
}

/** Evaluate trigger response (placeholder). */
export interface EvaluateResponse {
  accepted: boolean;
  checkRunId?: string;
  message?: string;
}

/** Proposal preparation request (placeholder). */
export interface ProposalRequest {
  skillId: number;
  namespace: string;
  slug: string;
  title: string;
  description: string;
  diffSummary?: VersionDiff;
}

/** Proposal preparation response (placeholder). */
export interface ProposalResponse {
  accepted: boolean;
  proposalId?: string;
  message?: string;
}
