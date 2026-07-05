import type { ReviewTaskView, PromotionRequestView } from "./frontend.js";

/** Request body for review/promotion approve and reject. */
export interface ReviewMutationRequest {
  comment?: string;
}

/** Response for review approve/reject. */
export interface ReviewMutationResponse {
  task: ReviewTaskView;
}

/** Request body for promotion approve and reject (same shape). */
export interface PromotionMutationRequest {
  comment?: string;
}

/** Response for promotion approve/reject. */
export interface PromotionMutationResponse {
  request: PromotionRequestView;
}

/** Response for review/promotion withdraw. */
export interface WithdrawResponse {
  status: string;
  version?: { id: number; version: string; status: string };
}
