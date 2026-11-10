import type { PublicPlan } from "@/modules/homepage/schemas/plan.schema";

/** Price per month, rounded to whole rupiah. */
export function monthlyPrice(plan: PublicPlan): number {
  return Math.round(plan.price / plan.durationMonths);
}

/**
 * The plan marked "Paling hemat" (pengguna, K2): the lowest price per month.
 * None when there is only one plan or when the cheapest per month is shared,
 * so the label never picks arbitrarily.
 */
export function cheapestPerMonth(plans: PublicPlan[]): PublicPlan | null {
  if (plans.length < 2) return null;
  const sorted = [...plans].sort((a, b) => a.price / a.durationMonths - b.price / b.durationMonths);
  const [first, second] = sorted;
  return first.price / first.durationMonths < second.price / second.durationMonths ? first : null;
}

/**
 * Whole-percent saving per month against the shortest plan, computed from the
 * data (never written by hand). Null when there is nothing to compare or no saving.
 */
export function savingPercent(plan: PublicPlan, plans: PublicPlan[]): number | null {
  const base = plans.reduce<PublicPlan | null>((min, p) => (!min || p.durationMonths < min.durationMonths ? p : min), null);
  if (!base || base === plan || base.durationMonths >= plan.durationMonths) return null;
  const pct = Math.floor((1 - plan.price / plan.durationMonths / (base.price / base.durationMonths)) * 100);
  return pct > 0 ? pct : null;
}
