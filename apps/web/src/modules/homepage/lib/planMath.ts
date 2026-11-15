import type { PublicPlan } from "@/modules/homepage/schemas/plan.schema";

/** How often a plan is billed, as the price line reads it: "per bulan", "per 6 bulan". */
export function billingPeriod(durationMonths: number): string {
  return durationMonths === 1 ? "per bulan" : `per ${durationMonths} bulan`;
}

/** The heading and lead of the price section, worded for the plans that are actually on sale. */
export interface PricingIntro {
  heading: string;
  lead: string;
}

const TRIAL = "Mulai dengan coba gratis 14 hari.";

/**
 * EP-14: the price section used to say "pilih lamanya" while a single monthly plan was on sale. The words now follow the
 * data: one plan, several plans of one length, or several lengths. Before the plans arrive (or when they cannot be read)
 * the heading promises nothing about them.
 */
export function pricingIntro(plans: PublicPlan[] | null): PricingIntro {
  if (!plans || plans.length === 0) {
    return { heading: "Harga terbuka", lead: `${TRIAL} Tanpa kartu kredit.` };
  }
  if (plans.length === 1) {
    const [plan] = plans;
    return {
      heading: "Harga terbuka, satu paket",
      lead: `${TRIAL} Setelah itu lanjutkan dengan paket ${plan.name}, ditagih ${billingPeriod(plan.durationMonths)}.`,
    };
  }
  if (new Set(plans.map((p) => p.durationMonths)).size === 1) {
    return {
      heading: "Harga terbuka, pilih paketnya",
      lead: `${TRIAL} Setelah itu pilih paket yang sesuai kebutuhan tim Anda; semuanya ditagih ${billingPeriod(plans[0].durationMonths)}.`,
    };
  }
  return { heading: "Harga terbuka, pilih lamanya", lead: `${TRIAL} Setelah itu pilih paket sesuai lama yang Anda butuhkan.` };
}

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
  // Integer arithmetic: 1 - 0.9 in floating point is 0.0999…, which floored would show 9% for a 10% saving.
  // saving = 1 - (plan.price / plan.months) / (base.price / base.months) = (den - plan.price * base.months) / den
  const den = base.price * plan.durationMonths;
  if (den <= 0) return null;
  const pct = Math.floor(((den - plan.price * base.durationMonths) * 100) / den);
  return pct > 0 ? pct : null;
}
