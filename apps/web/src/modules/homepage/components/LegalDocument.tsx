import { useEffect, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { ChevronDown } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Container } from "@/modules/homepage/components/Container";
import { cn } from "@/shared/lib/cn";

export interface LegalSection {
  id: string;
  title: string;
  body: string[];
}

interface LegalDocumentProps {
  title: string;
  updated: string;
  /** Optional lead paragraph under the title. */
  intro?: ReactNode;
  sections: LegalSection[];
  /** Heading and text of the closing contact note; defaults to the general one. */
  closing?: { title: string; body: ReactNode };
}

const tocLinkClass = (active: boolean) =>
  cn(
    "flex min-h-11 items-center border-l-2 py-1.5 pl-4 text-[14.5px] leading-snug outline-hidden focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-2 focus-visible:outline-ring",
    active
      ? "border-primary font-semibold text-foreground"
      : "border-border text-muted-foreground hover:border-input hover:text-foreground",
  );

/**
 * Marks the section being read in the table of contents. Only the marker
 * depends on the observer: every section is always rendered and visible.
 */
function useActiveSection(ids: string[]) {
  const [activeId, setActiveId] = useState("");
  useEffect(() => {
    if (typeof IntersectionObserver === "undefined") return;
    const elements = ids.map((id) => document.getElementById(id)).filter((el): el is HTMLElement => el !== null);
    // The section being read is the last one whose top has passed 30% of the viewport (the first one near the top).
    const mark = () => {
      const line = window.innerHeight * 0.3;
      let current = elements[0]?.id ?? "";
      for (const el of elements) if (el.getBoundingClientRect().top <= line) current = el.id;
      setActiveId(current);
    };
    const observer = new IntersectionObserver(mark, { threshold: [0, 1], rootMargin: "0px 0px -70% 0px" });
    for (const el of elements) observer.observe(el);
    mark();
    return () => observer.disconnect();
  }, [ids]);
  return activeId;
}

/**
 * Shared reading layout of the legal pages (Syarat & Ketentuan, Kebijakan
 * Privasi, Perjanjian Pemrosesan Data, Kebijakan Refund): a quiet title block,
 * a table of contents (sticky on the left on desktop, a disclosure above the
 * text on phones), numbered sections in a ≤70-character column, and a contact
 * note. No entrance animation (ADR-0038): legal text is never hidden.
 */
export function LegalDocument({ title, updated, intro, sections, closing }: LegalDocumentProps) {
  const [ids] = useState(() => sections.map((s) => s.id));
  const activeId = useActiveSection(ids);

  const links = () => (
    <ol className="flex flex-col">
      {sections.map((section) => (
        <li key={section.id}>
          <a
            href={`#${section.id}`}
            aria-current={activeId === section.id ? "location" : undefined}
            className={tocLinkClass(activeId === section.id)}
          >
            {section.title}
          </a>
        </li>
      ))}
    </ol>
  );

  return (
    <div className="bg-background">
      <header className="border-b border-border bg-muted">
        <Container className="py-12 md:py-16">
          <div className="max-w-[44rem]">
            <h1 className="text-[34px] font-extrabold leading-[1.04] tracking-[-0.035em] text-foreground md:text-[48px]">
              {title}
            </h1>
            <p className="mt-4 text-[15px] text-muted-foreground">Terakhir diperbarui: {updated}</p>
            {intro && <div className="mt-5 max-w-[34rem] text-[17px] leading-relaxed text-foreground">{intro}</div>}
          </div>
        </Container>
      </header>

      <Container className="py-10 md:py-16">
        <div className="md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-12 lg:grid-cols-[260px_minmax(0,1fr)] lg:gap-20">
          <nav aria-label="Daftar isi" className="hidden md:block">
            <div className="sticky top-24">
              <p className="mb-3 text-[14px] font-semibold text-foreground">Daftar isi</p>
              {links()}
            </div>
          </nav>

          <details className="group mb-10 rounded-xl border border-border bg-muted md:hidden">
            <summary className="flex min-h-12 cursor-pointer list-none items-center justify-between gap-3 rounded-xl px-4 text-[15px] font-semibold text-foreground outline-hidden focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-2 focus-visible:outline-ring [&::-webkit-details-marker]:hidden">
              Daftar isi
              <ChevronDown aria-hidden className="h-4 w-4 text-muted-foreground motion-safe:transition-transform group-open:rotate-180" />
            </summary>
            <nav aria-label="Daftar isi" className="px-4 pb-3">
              {links()}
            </nav>
          </details>

          <article className="max-w-[34rem]">
            <div className="flex flex-col gap-11">
              {sections.map((section) => (
                <section key={section.id} id={section.id} aria-labelledby={`${section.id}-judul`} className="scroll-mt-24">
                  <h2
                    id={`${section.id}-judul`}
                    className="text-[21px] font-extrabold leading-snug tracking-[-0.03em] text-foreground md:text-[23px]"
                  >
                    {section.title}
                  </h2>
                  <div className="mt-4 flex flex-col gap-4">
                    {section.body.map((paragraph, i) => (
                      <p key={i} className="text-[16.5px] leading-[1.75] text-foreground md:text-[17px]">
                        {paragraph}
                      </p>
                    ))}
                  </div>
                </section>
              ))}
            </div>

            <aside className="mt-14 border-t-2 border-seal pt-6">
              <h2 className="text-[18px] font-bold tracking-[-0.02em] text-foreground">
                {closing?.title ?? "Ada pertanyaan tentang dokumen ini?"}
              </h2>
              <p className="mt-2 max-w-[32rem] text-[16px] leading-relaxed text-muted-foreground">
                {closing?.body ?? (
                  <>
                    Hubungi tim kami melalui halaman{" "}
                    <Link to={ROUTE_PATHS.homepageContact} className="font-semibold text-foreground underline underline-offset-4 hover:text-primary-hover">
                      Kontak
                    </Link>
                    .
                  </>
                )}
              </p>
            </aside>
          </article>
        </div>
      </Container>
    </div>
  );
}
