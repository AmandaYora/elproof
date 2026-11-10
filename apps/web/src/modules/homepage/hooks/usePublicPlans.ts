import { useEffect, useState } from "react";
import { fetchPublicPlans } from "@/modules/homepage/services/plans.service";
import type { PublicPlan } from "@/modules/homepage/schemas/plan.schema";

export type PublicPlansState =
  | { status: "loading" }
  | { status: "ready"; plans: PublicPlan[] }
  /** The API failed, answered in another shape, or has no active plan: the page asks people to use WhatsApp. */
  | { status: "unavailable" };

/** ElProof's active public plans for the price section. The request is cancelled when the page unmounts. */
export function usePublicPlans(): PublicPlansState {
  const [state, setState] = useState<PublicPlansState>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();
    fetchPublicPlans(controller.signal)
      .then((plans) => setState(plans.length > 0 ? { status: "ready", plans } : { status: "unavailable" }))
      .catch(() => {
        if (!controller.signal.aborted) setState({ status: "unavailable" });
      });
    return () => controller.abort();
  }, []);

  return state;
}
