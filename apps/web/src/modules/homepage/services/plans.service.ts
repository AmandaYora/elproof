import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { publicPlanListSchema, type PublicPlan } from "@/modules/homepage/schemas/plan.schema";

/** ElProof's active public plans, shortest first. Throws when the API is unreachable or the shape is wrong. */
export async function fetchPublicPlans(signal?: AbortSignal): Promise<PublicPlan[]> {
  const res = await httpClient.get(API.public.plans, { signal });
  const plans = publicPlanListSchema.parse(res.data.data);
  return [...plans].sort((a, b) => a.durationMonths - b.durationMonths || a.price - b.price);
}
