import type { SkillHubTransport, Envelope } from "../core.js";
import type { PromotionMutationRequest, PromotionMutationResponse, WithdrawResponse } from "../types/mutations.js";

export function approvePromotion(
  transport: SkillHubTransport,
  id: number,
  req?: PromotionMutationRequest,
): Promise<Envelope<PromotionMutationResponse>> {
  return transport.request(`/api/v1/promotions/${transport.path(id)}/approve`, {
    method: "POST",
    body: req ? JSON.stringify(req) : "{}",
  });
}

export function rejectPromotion(
  transport: SkillHubTransport,
  id: number,
  req?: PromotionMutationRequest,
): Promise<Envelope<PromotionMutationResponse>> {
  return transport.request(`/api/v1/promotions/${transport.path(id)}/reject`, {
    method: "POST",
    body: req ? JSON.stringify(req) : "{}",
  });
}

export function withdrawPromotion(
  transport: SkillHubTransport,
  id: number,
): Promise<Envelope<WithdrawResponse>> {
  return transport.request(`/api/v1/promotions/${transport.path(id)}/withdraw`, {
    method: "POST",
    body: "{}",
  });
}
