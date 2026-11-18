import { Link } from "react-router-dom";
import { Check, MessageCircle } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { ctaClass } from "@/modules/homepage/components/cta";
import { whatsappUrl } from "@/modules/homepage/data/contact";
import { usePublicPlans, type PublicPlansState } from "@/modules/homepage/hooks/usePublicPlans";
import { cheapestPerMonth, monthlyPrice, pricingIntro, savingPercent } from "@/modules/homepage/lib/planMath";
import type { PublicPlan } from "@/modules/homepage/schemas/plan.schema";
import { formatCurrency } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";

const WHATSAPP_PRICE_URL = whatsappUrl("Halo ElProof, saya ingin tahu harga paket langganan.");

/** What every plan includes, whatever the admin wrote as features (true for all plans today). */
const ALWAYS = ["Coba gratis 14 hari, tanpa kartu kredit", "WO Console untuk tim dan Portal Klien untuk pasangan", "Dibayar per periode dari menu Langganan"];

function PlanCard({ plan, plans, best, featured }: { plan: PublicPlan; plans: PublicPlan[]; best: boolean; featured: boolean }) {
  const saving = savingPercent(plan, plans);
  const shortest = plans.reduce((min, p) => (p.durationMonths < min.durationMonths ? p : min), plans[0]);
  const items = [...plan.features, ...ALWAYS];
  return (
    <li
      className={cn(
        "flex flex-col rounded-[20px] border p-6 sm:p-7",
        featured ? "border-primary bg-primary text-primary-foreground" : "border-border bg-card",
      )}
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <h3 className="text-[22px] leading-tight font-bold tracking-[-0.02em]">{plan.name}</h3>
        {best && plans.length > 1 && (
          <span
            className={cn(
              "rounded-full border px-2.5 py-1 text-[12.5px] font-semibold",
              featured ? "border-seal-on-primary/60 text-seal-on-primary" : "border-seal bg-seal-soft text-seal-text",
            )}
          >
            Paling hemat
          </span>
        )}
      </div>
      <p className="mt-5 text-[36px] leading-none font-bold tracking-[-0.025em] tabular-nums">{formatCurrency(plan.price)}</p>
      <p className={cn("mt-2 text-[14px]", featured ? "text-on-primary-muted" : "text-muted-foreground")}>
        {plan.durationMonths > 1 ? (
          <>
            per {plan.durationMonths} bulan, setara <span className="tabular-nums">{formatCurrency(monthlyPrice(plan))}</span> per bulan
          </>
        ) : (
          "per bulan"
        )}
      </p>
      {saving !== null && (
        <p className={cn("mt-1 text-[14px] font-semibold", featured ? "text-seal-on-primary" : "text-seal-text")}>
          Hemat {saving}% dibanding paket {shortest.durationMonths} bulan
        </p>
      )}
      <ul className="mt-6 grid gap-2.5 text-[15px]">
        {items.map((item) => (
          <li key={item} className="flex items-start gap-2.5">
            <Check aria-hidden className={cn("mt-1 size-4 shrink-0", featured ? "text-seal-on-primary" : "text-seal")} strokeWidth={2.6} />
            {item}
          </li>
        ))}
      </ul>
      <Link
        to={`${ROUTE_PATHS.signup}?paket=${plan.id}`}
        aria-label={`Pilih paket ${plan.name}`}
        className={ctaClass(featured ? "inverted" : "outline", "md", "mt-8 w-full")}
      >
        Pilih paket
      </Link>
    </li>
  );
}

/** One plan on sale: the card, and beside it what happens after signing up, so the section is not a lonely box. */
function SinglePlan({ plan, plans }: { plan: PublicPlan; plans: PublicPlan[] }) {
  return (
    <div className="grid items-start gap-8 lg:grid-cols-[minmax(0,420px)_1fr] lg:gap-14">
      <ul>
        <PlanCard plan={plan} plans={plans} best featured />
      </ul>
      <div className="lg:pt-4">
        <h3 className="text-[20px] font-bold tracking-[-0.02em]">Setelah Anda mendaftar</h3>
        <ol className="mt-5 grid gap-5">
          {[
            ["Workspace siap dalam semenit", "Daftar dengan akun Google, isi nama usaha, dan undang tim Anda."],
            ["14 hari memakai semua fitur", "Buat proyek sungguhan, kirim tautan Portal Klien ke pasangan lewat WhatsApp."],
            [`Lanjut dengan paket ${plan.name}`, "Bayar dari menu Langganan bila cocok. Masa langganan dimulai setelah masa coba habis."],
          ].map(([title, body], i) => (
            <li key={title} className="flex gap-4">
              <span className="grid size-8 shrink-0 place-items-center rounded-full border border-border text-[14px] font-bold tabular-nums">{i + 1}</span>
              <div>
                <p className="font-semibold">{title}</p>
                <p className="mt-0.5 text-[15px] text-muted-foreground">{body}</p>
              </div>
            </li>
          ))}
        </ol>
        <a href={WHATSAPP_PRICE_URL} target="_blank" rel="noreferrer" className="mt-7 inline-flex items-center gap-2 rounded-md text-[15px] font-semibold underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
          <MessageCircle aria-hidden className="size-4" /> Masih ragu? Tanya dulu lewat WhatsApp
        </a>
      </div>
    </div>
  );
}

function SkeletonCards() {
  return (
    <div aria-busy="true">
      <p className="sr-only">Memuat harga paket</p>
      <ul aria-hidden className="grid gap-5 md:grid-cols-3">
        {[0, 1, 2].map((i) => (
          <li key={i} className="flex flex-col gap-4 rounded-[20px] border border-border p-7">
            <span className="h-6 w-32 rounded-md bg-muted motion-safe:animate-pulse" />
            <span className="h-10 w-44 rounded-md bg-muted motion-safe:animate-pulse" />
            <span className="h-4 w-full rounded-md bg-muted motion-safe:animate-pulse" />
            <span className="h-4 w-4/5 rounded-md bg-muted motion-safe:animate-pulse" />
            <span className="mt-4 h-12 w-full rounded-full bg-muted motion-safe:animate-pulse" />
          </li>
        ))}
      </ul>
    </div>
  );
}

/**
 * The price section: heading, lead and the list, read from the active public plans. No price is ever written by hand,
 * and the heading follows what is on sale (pricingIntro).
 */
export function Pricing({ headingClassName, leadClassName }: { headingClassName: string; leadClassName: string }) {
  const state = usePublicPlans();
  const intro = pricingIntro(state.status === "ready" ? state.plans : null);

  return (
    <>
      <div className="max-w-[640px]">
        <h2 className={headingClassName}>{intro.heading}</h2>
        <p className={leadClassName}>{intro.lead}</p>
      </div>
      <div className="mt-10">
        <PlanList state={state} />
      </div>
    </>
  );
}

function PlanList({ state }: { state: PublicPlansState }) {
  if (state.status === "loading") return <SkeletonCards />;

  if (state.status === "unavailable") {
    return (
      <div className="flex flex-col gap-4 rounded-[20px] border border-border p-7 sm:flex-row sm:items-center sm:justify-between">
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
      {state.plans.length === 1 ? (
        <SinglePlan plan={state.plans[0]} plans={state.plans} />
      ) : (
        <ul className={cn("grid gap-5", state.plans.length === 2 ? "md:grid-cols-2" : "md:grid-cols-3")}>
          {state.plans.map((plan) => (
            <PlanCard key={plan.id} plan={plan} plans={state.plans} best={plan === best} featured={plan === best} />
          ))}
        </ul>
      )}
      <p className="mt-6 text-[13.5px] text-muted-foreground">Harga dalam Rupiah, termasuk semua fitur paket yang tercantum.</p>
    </>
  );
}
