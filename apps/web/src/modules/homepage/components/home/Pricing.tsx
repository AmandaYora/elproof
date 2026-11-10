import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { ctaClass } from "@/modules/homepage/components/cta";
import { CONTACT } from "@/modules/homepage/data/contact";
import { usePublicPlans } from "@/modules/homepage/hooks/usePublicPlans";
import { cheapestPerMonth, monthlyPrice, savingPercent } from "@/modules/homepage/lib/planMath";
import type { PublicPlan } from "@/modules/homepage/schemas/plan.schema";
import { formatCurrency } from "@/shared/lib/formatters";

const WHATSAPP_PRICE_URL = `https://wa.me/${CONTACT.phones[0].number}?text=${encodeURIComponent("Halo ElProof, saya ingin tahu harga paket langganan.")}`;

const ROW = "grid gap-x-6 gap-y-3 border-b border-border py-7 md:grid-cols-[1.1fr_1.4fr_0.95fr_auto] md:items-center";

function PlanRow({ plan, plans, best }: { plan: PublicPlan; plans: PublicPlan[]; best: boolean }) {
  const saving = savingPercent(plan, plans);
  const shortest = plans.reduce((min, p) => (p.durationMonths < min.durationMonths ? p : min), plans[0]);
  return (
    <li className={ROW}>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <h3 className="text-[22px] leading-tight font-bold tracking-[-0.02em]">{plan.name}</h3>
        {best && (
          <span className="rounded-full border border-seal bg-seal-soft px-2.5 py-1 text-[12.5px] font-semibold text-seal-text">Paling hemat</span>
        )}
      </div>
      {plan.features.length > 0 ? (
        <p className="text-[14.5px] text-muted-foreground">{plan.features.join(", ")}</p>
      ) : (
        <span className="hidden md:block" />
      )}
      <div>
        <p className="text-[28px] leading-[1.1] font-extrabold tracking-[-0.03em] tabular-nums">{formatCurrency(plan.price)}</p>
        <p className="mt-1 text-[13.5px] text-muted-foreground">
          {plan.durationMonths > 1 ? (
            <>
              <span className="tabular-nums">{formatCurrency(monthlyPrice(plan))}</span> per bulan
            </>
          ) : (
            "ditagih bulanan"
          )}
        </p>
        {saving !== null && (
          <p className="mt-0.5 text-[13.5px] font-semibold text-seal-text">
            Hemat {saving}% dibanding paket {shortest.durationMonths} bulan
          </p>
        )}
      </div>
      <Link
        to={`${ROUTE_PATHS.signup}?paket=${plan.id}`}
        aria-label={`Pilih paket ${plan.name}`}
        className={ctaClass(best ? "primary" : "outline", "md", "w-full md:w-auto")}
      >
        Pilih paket
      </Link>
    </li>
  );
}

function SkeletonRows() {
  return (
    <div aria-busy="true">
      <p className="sr-only">Memuat harga paket</p>
      <ul aria-hidden className="border-t border-foreground">
        {[0, 1, 2].map((i) => (
          <li key={i} className={ROW}>
            <span className="h-6 w-36 rounded-md bg-muted motion-safe:animate-pulse" />
            <span className="h-4 w-full max-w-72 rounded-md bg-muted motion-safe:animate-pulse" />
            <span className="h-8 w-40 rounded-md bg-muted motion-safe:animate-pulse" />
            <span className="h-12 w-full rounded-full bg-muted motion-safe:animate-pulse md:w-32" />
          </li>
        ))}
      </ul>
    </div>
  );
}

/** The price list, read from the active public plans. No price is ever written by hand. */
export function Pricing() {
  const state = usePublicPlans();

  if (state.status === "loading") return <SkeletonRows />;

  if (state.status === "unavailable") {
    return (
      <div className="flex flex-col gap-4 border-y border-foreground py-8 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-[17px] font-semibold">Harga sedang diperbarui. Tanyakan lewat WhatsApp.</p>
        <a href={WHATSAPP_PRICE_URL} target="_blank" rel="noreferrer" className={ctaClass("outline", "md", "w-full sm:w-auto")}>
          Tanya harga lewat WhatsApp
        </a>
      </div>
    );
  }

  const best = cheapestPerMonth(state.plans);
  return (
    <>
      <ul className="border-t border-foreground">
        {state.plans.map((plan) => (
          <PlanRow key={plan.id} plan={plan} plans={state.plans} best={plan === best} />
        ))}
      </ul>
      <p className="mt-5 text-[13.5px] text-muted-foreground">Harga dalam Rupiah, termasuk semua fitur paket yang tercantum.</p>
    </>
  );
}
