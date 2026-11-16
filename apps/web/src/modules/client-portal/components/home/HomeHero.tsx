import { useState } from "react";
import { cn } from "@/shared/lib/cn";
import { formatDate } from "@/shared/lib/formatters";
import { countdown } from "@/modules/client-portal/lib/stage";
import { useCoverImage } from "@/modules/client-portal/lib/useCoverImage";
import type { PortalHome } from "@/modules/client-portal/types";

/**
 * The seal from the logo (ring, inner ring, check), drawn in thin gold lines on the navy hero when there is no photo.
 * Decoration only; it never stands for proof or status.
 */
function SealMotif() {
  return (
    <svg
      aria-hidden
      viewBox="0 0 200 200"
      fill="none"
      className="pointer-events-none absolute -top-14 -right-16 size-64 text-seal-on-primary opacity-20 sm:-top-20 sm:-right-10 sm:size-80"
    >
      <circle cx="100" cy="100" r="96" stroke="currentColor" strokeWidth="1" />
      <circle cx="100" cy="100" r="80" stroke="currentColor" strokeWidth="1" strokeDasharray="2 5" />
      <circle cx="100" cy="100" r="62" stroke="currentColor" strokeWidth="1.5" />
      <path d="M72 101 L92 120 L130 80" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

/**
 * The couple's names and the countdown: the one loud thing on the page. With the WO's cover photo the picture keeps
 * its own colours and navy sits only under the text (as on the sign-in screen); without one it is a navy band with a
 * faint gold seal. Names and countdown in heavy, tight type, divided by a seal-gold rule (DESIGN.md §5).
 */
export function HomeHero({ project, daysToEvent }: { project: PortalHome["project"]; daysToEvent: number }) {
  const c = countdown(project.status, daysToEvent);
  const cover = useCoverImage(project.coverUrl);
  const [loaded, setLoaded] = useState(false);
  const [broken, setBroken] = useState(false);
  const photo = cover && !broken ? cover : null;

  return (
    <section
      aria-labelledby="hero-names"
      className={cn(
        "relative isolate flex flex-col justify-end overflow-hidden rounded-3xl bg-primary px-6 py-8 text-primary-foreground sm:px-10 sm:py-12",
        photo && "min-h-[23rem] sm:min-h-[26rem]",
      )}
    >
      {photo ? (
        <>
          <img
            src={photo}
            alt=""
            loading="lazy"
            decoding="async"
            onLoad={() => setLoaded(true)}
            onError={() => setBroken(true)}
            className={cn("absolute inset-0 -z-10 size-full object-cover motion-safe:transition-opacity motion-safe:duration-500", loaded ? "opacity-100" : "opacity-0")}
          />
          <div aria-hidden className="pointer-events-none absolute inset-0 -z-10 bg-gradient-to-t from-primary from-20% via-primary/70 via-45% to-transparent to-75%" />
        </>
      ) : (
        <SealMotif />
      )}

      <p className="text-[13px] font-medium text-on-primary-muted">
        {formatDate(project.eventDate)}
        {project.venue ? ` · ${project.venue}` : ""}
      </p>
      <h1 id="hero-names" className="mt-2 text-[28px] leading-[1.08] font-extrabold tracking-[-0.03em] text-balance sm:text-[40px]">
        {project.brideName} &amp; {project.groomName}
      </h1>
      <div className="mt-6 flex flex-wrap items-end gap-x-4 gap-y-1 border-t border-seal-on-primary pt-5">
        <span className="text-[56px] leading-none font-extrabold tracking-[-0.035em] tabular-nums sm:text-[72px]">{c.figure}</span>
        <span className="max-w-[16rem] pb-1.5 text-[14px] leading-snug text-on-primary-muted">{c.caption}</span>
      </div>
    </section>
  );
}
