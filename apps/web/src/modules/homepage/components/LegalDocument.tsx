import { useEffect, useState, type ComponentType, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Reveal } from "@/modules/homepage/components/Reveal";
import { cn } from "@/shared/lib/cn";

export interface LegalSection {
  id: string;
  title: string;
  body: string[];
}

interface LegalDocumentProps {
  icon: ComponentType<{ className?: string }>;
  title: string;
  updated: string;
  /** Optional lead paragraph under the title. */
  intro?: ReactNode;
  sections: LegalSection[];
}

/**
 * Shared frame for the legal pages (Syarat & Ketentuan, Kebijakan Privasi,
 * Perjanjian Pemrosesan Data): title, table of contents that follows the
 * reader, numbered sections, and a contact note.
 */
export function LegalDocument({ icon: Icon, title, updated, intro, sections }: LegalDocumentProps) {
  const [activeId, setActiveId] = useState("");

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) setActiveId(entry.target.id);
        });
      },
      { rootMargin: "-20% 0px -80% 0px" },
    );
    sections.forEach((s) => {
      const el = document.getElementById(s.id);
      if (el) observer.observe(el);
    });
    return () => observer.disconnect();
  }, [sections]);

  return (
    <div className="min-h-screen bg-background">
      <section className="relative overflow-hidden border-b border-border bg-white px-6 py-20 md:py-24">
        <div className="absolute inset-0 bg-[linear-gradient(to_right,#f1f5f9_1px,transparent_1px),linear-gradient(to_bottom,#f1f5f9_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)] opacity-60"></div>
        <div className="relative z-10 mx-auto max-w-4xl text-center">
          <Reveal>
            <div className="inline-flex items-center gap-2 rounded-full bg-blue-50 px-3.5 py-1.5 text-[13px] font-semibold text-blue-600">
              <Icon className="h-4 w-4" /> Dokumen layanan
            </div>
            <h1 className="mt-6 font-display text-[36px] font-semibold tracking-tight text-navy-950 md:text-[48px]">{title}</h1>
            <p className="mt-4 text-[15px] font-medium text-text-secondary">Terakhir diperbarui: {updated}</p>
            {intro && <div className="mx-auto mt-6 max-w-2xl text-[15.5px] leading-relaxed text-text-secondary">{intro}</div>}
          </Reveal>
        </div>
      </section>

      <div className="mx-auto max-w-6xl px-4 py-16 sm:px-6 md:py-24">
        <div className="flex flex-col gap-12 md:flex-row lg:gap-20">
          <nav aria-label="Daftar isi" className="hidden w-64 shrink-0 md:block">
            <div className="sticky top-32 flex flex-col gap-2">
              <h3 className="mb-4 text-[13px] font-semibold text-text-secondary">Daftar isi</h3>
              {sections.map((section) => (
                <a
                  key={section.id}
                  href={`#${section.id}`}
                  className={cn(
                    "border-l-2 py-1.5 pl-4 text-[14px] font-medium transition-all duration-200",
                    activeId === section.id
                      ? "rounded-r-md border-navy-900 bg-navy-50/50 text-navy-950"
                      : "border-border text-text-secondary hover:border-navy-400 hover:text-navy-900",
                  )}
                >
                  {section.title}
                </a>
              ))}
            </div>
          </nav>

          <div className="max-w-3xl flex-1 rounded-3xl border border-border bg-white p-6 shadow-sm sm:p-8 md:p-12">
            <Reveal className="flex flex-col gap-12">
              {sections.map((section) => (
                <div key={section.id} id={section.id} className="scroll-mt-32">
                  <h2 className="text-[20px] font-bold tracking-tight text-navy-950">{section.title}</h2>
                  <div className="mt-4 flex flex-col gap-4">
                    {section.body.map((paragraph, i) => (
                      <p key={i} className="text-[15.5px] leading-relaxed text-text-secondary">
                        {paragraph}
                      </p>
                    ))}
                  </div>
                </div>
              ))}

              <div className="mt-8 rounded-2xl border border-border bg-surface-muted/60 p-6 sm:p-8">
                <h3 className="text-[16px] font-bold text-navy-950">Ada pertanyaan tentang dokumen ini?</h3>
                <p className="mt-3 text-[14.5px] leading-relaxed text-text-secondary">
                  Hubungi tim kami melalui halaman{" "}
                  <Link to={ROUTE_PATHS.homepageContact} className="font-semibold text-navy-900 hover:underline">
                    Kontak
                  </Link>
                  .
                </p>
              </div>
            </Reveal>
          </div>
        </div>
      </div>
    </div>
  );
}
