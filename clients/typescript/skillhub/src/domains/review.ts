import type { SkillHubTransport, Envelope } from "../core.js";
import type { ReviewMutationRequest, ReviewMutationResponse, WithdrawResponse } from "../types/mutations.js";

export function approveReview(
  transport: SkillHubTransport,
  id: number,
  req?: ReviewMutationRequest,
): Promise<Envelope<ReviewMutationResponse>> {
  return transport.request(`/api/v1/reviews/${transport.path(id)}/approve`, {
    method: "POST",
    body: req ? JSON.stringify(req) : "{}",
  });
}

export function rejectReview(
  transport: SkillHubTransport,
  id: number,
  req?: ReviewMutationRequest,
): Promise<Envelope<ReviewMutationResponse>> {
  return transport.request(`/api/v1/reviews/${transport.path(id)}/reject`, {
    method: "POST",
    body: req ? JSON.stringify(req) : "{}",
  });
}

export function withdrawReview(
  transport: SkillHubTransport,
  id: number,
): Promise<Envelope<WithdrawResponse>> {
  return transport.request(`/api/v1/reviews/${transport.path(id)}/withdraw`, {
    method: "POST",
    body: "{}",
  });
}
