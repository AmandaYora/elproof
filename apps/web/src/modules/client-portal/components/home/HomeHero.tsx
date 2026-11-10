import { formatDate } from "@/shared/lib/formatters";
import { countdown } from "@/modules/client-portal/lib/stage";
import type { PortalHome } from "@/modules/client-portal/types";

/**
 * The couple's names and the countdown. The one loud thing on the page: a plum
 * band with the names and the countdown figure in heavy, tight type, divided by
 * a thin seal-gold rule (DESIGN.md §5); everything below it stays quiet.
 */
export function HomeHero({ project, daysToEvent }: { project: PortalHome["project"]; daysToEvent: number }) {
  const c = countdown(project.status, daysToEvent);
  return (
    <section aria-labelledby="hero-names" className="overflow-hidden rounded-3xl bg-primary px-6 py-8 text-primary-foreground sm:px-10 sm:py-12">
      <p className="text-[13px] font-medium text-on-primary-muted">{formatDate(project.eventDate)}{project.venue ? ` · ${project.venue}` : ""}</p>
      <h1 id="hero-names" className="mt-2 text-[28px] font-extrabold leading-[1.08] tracking-[-0.03em] sm:text-[40px]">
        {project.brideName} &amp; {project.groomName}
      </h1>
      <div className="mt-6 flex flex-wrap items-end gap-x-4 gap-y-1 border-t border-seal/70 pt-5">
        <span className="text-[56px] font-extrabold leading-none tracking-[-0.035em] tabular-nums sm:text-[72px]">{c.figure}</span>
        <span className="max-w-[16rem] pb-1.5 text-[14px] leading-snug text-on-primary-muted">{c.caption}</span>
      </div>
    </section>
  );
}
