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

/**
 * The active public plan a visitor picked with "Pilih paket" (/daftar?paket=<id>, EP-14). Null while loading, when the id
 * is not a plan id, when no active plan has it, or when the API cannot be read: Daftar is then the plain trial signup.
 * Starts as null on every render path, so the prerendered Daftar (no query string) hydrates without a mismatch.
 */
export function useChosenPlan(id: string | null): PublicPlan | null {
  const [found, setFound] = useState<{ id: string; plan: PublicPlan } | null>(null);

  useEffect(() => {
    if (!id || !/^[1-9]\d{0,9}$/.test(id)) return;
    const controller = new AbortController();
    fetchPublicPlans(controller.signal)
      .then((plans) => {
        const plan = plans.find((p) => String(p.id) === id);
        setFound(plan ? { id, plan } : null);
      })
      .catch(() => {
        // Unknown or unreachable: the page stays the plain trial signup.
      });
    return () => controller.abort();
  }, [id]);

  return found && found.id === id ? found.plan : null;
}
