/**
 * Core transport, error handling, and shared types for the SkillHub SDK.
 *
 * These are implementation details used by domain modules; the public
 * API surface is re-exported from src/index.ts.
 */

// ── Configuration ───────────────────────────────────────────────────────

/** Options for constructing a SkillHubClient. */
export interface SkillHubClientOptions {
  /** Base URL of the SkillHub server (default: http://localhost:8080). */
  baseUrl?: string;
  /** Custom fetch implementation (for SSR, tests, or polyfills). */
  fetch?: typeof fetch;
  /** Request credentials mode (e.g. "include" for cookie-based auth). */
  credentials?: RequestCredentials;
  /** Static bearer token sent as Authorization: Bearer <token>. */
  token?: string;
  /** Dynamic token provider called before each request. */
  getToken?: () => string | undefined | Promise<string | undefined>;
  /** Additional headers merged into every request. */
  headers?: HeadersInit;
}

/** Constructor argument: a base URL string or an options object. */
export type SkillHubClientConfig = string | SkillHubClientOptions;

// ── Error handling ──────────────────────────────────────────────────────

/** Typed error thrown by unwrap() and carried by failed envelopes. */
export class SkillHubError extends Error {
  readonly code: string;
  readonly status?: number;
  readonly details?: unknown;
  readonly response?: Response;

  constructor(
    code: string,
    message: string,
    status?: number,
    details?: unknown,
    response?: Response,
  ) {
    super(message);
    this.name = "SkillHubError";
    this.code = code;
    this.status = status;
    this.details = details;
    this.response = response;
  }
}

// ── Envelope ────────────────────────────────────────────────────────────

/** Standard response envelope shared by every SkillHub HTTP endpoint. */
export interface Envelope<T = unknown> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string; details?: unknown };
  /** HTTP status code (attached by the client after each request). */
  status?: number;
}

// ── Transport interface ─────────────────────────────────────────────────

/**
 * Internal transport interface used by domain endpoint functions.
 * SkillHubClient implements this interface; domain functions accept
 * it as their first argument.
 *
 * The `request` method mirrors the original SkillHubClient.fetch:
 * it accepts a path and optional RequestInit, and returns Envelope<T>.
 * The method is specified via init.method (defaults to GET when absent).
 */
export interface SkillHubTransport {
  path(...parts: Array<string | number>): string;
  query(params: Record<string, string | number | boolean | string[] | undefined>): string;
  request<T>(path: string, init?: RequestInit): Promise<Envelope<T>>;
}

// ── Transport helpers (used by SkillHubClient) ──────────────────────────

/** Build a URL-safe path from segments, encoding each one. */
export function encodePath(...parts: Array<string | number>): string {
  return parts.map((p) => encodeURIComponent(String(p))).join("/");
}

/** Build a query string from a params record. */
export function encodeQuery(
  params: Record<string, string | number | boolean | string[] | undefined>,
): string {
  const sp = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined) continue;
    if (Array.isArray(value)) {
      sp.set(key, value.join(","));
    } else {
      sp.set(key, String(value));
    }
  }
  const qs = sp.toString();
  return qs ? `?${qs}` : "";
}
