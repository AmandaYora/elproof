import { formatDate } from "@/shared/lib/formatters";
import { countdown } from "@/modules/client-portal/lib/stage";
import type { PortalHome } from "@/modules/client-portal/types";

/**
 * The couple's names and the countdown. The one loud thing on the page: names
 * and the countdown figure are set in the display serif on the workspace's
 * brand color; everything below it stays quiet.
 */
export function HomeHero({ project, daysToEvent }: { project: PortalHome["project"]; daysToEvent: number }) {
  const c = countdown(project.status, daysToEvent);
  return (
    <section aria-labelledby="hero-names" className="overflow-hidden rounded-3xl bg-navy-900 px-6 py-8 text-white sm:px-10 sm:py-12">
      <p className="text-[13px] font-medium text-white/80">{formatDate(project.eventDate)}{project.venue ? ` · ${project.venue}` : ""}</p>
      <h1 id="hero-names" className="mt-2 font-display text-[28px] font-semibold leading-tight tracking-tight sm:text-[40px]">
        {project.brideName} &amp; {project.groomName}
      </h1>
      <div className="mt-6 flex flex-wrap items-end gap-x-4 gap-y-1 border-t border-white/20 pt-5">
        <span className="font-display text-[56px] font-semibold leading-none tabular-nums sm:text-[72px]">{c.figure}</span>
        <span className="max-w-[16rem] pb-1.5 text-[14px] leading-snug text-white/85">{c.caption}</span>
      </div>
    </section>
  );
}
