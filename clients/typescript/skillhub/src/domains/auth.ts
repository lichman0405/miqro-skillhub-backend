import type { SkillHubTransport, Envelope } from "../core.js";
import type { Principal } from "../types/common.js";

export function login(
  transport: SkillHubTransport,
  username: string,
  password: string,
): Promise<Envelope<Principal>> {
  return transport.request<Principal>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
}

export function me(transport: SkillHubTransport): Promise<Envelope<Principal>> {
  return transport.request("/api/v1/auth/me", { credentials: "include" });
}
