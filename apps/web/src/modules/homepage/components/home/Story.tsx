import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { STORY_STEPS } from "@/modules/homepage/components/home/storySteps";
import { MockItem, PhoneFrame, StatusChip } from "@/modules/homepage/components/home/mock";
import { formatCurrency } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";

const DESKTOP = "(min-width: 1024px)";

/** What the phone shows at each step of the story (example data). */
const SCREENS: { heading: string; body: ReactNode }[] = [
  {
    heading: "Kontrak disepakati",
    body: (
      <>
        <MockItem title="Paket Gold, 400 tamu" detail="Kontrak ditandatangani 15 Jun 2026" />
        <MockItem title="3 termin dibuat" detail="Uang muka, termin 2, dan pelunasan" />
        <MockItem title="24 tahapan dari templat" detail="Tanggal mengikuti hari H" />
      </>
    ),
  },
  {
    heading: "Vendor terkunci",
    body: (
      <>
        <MockItem title="Katering Rasa Nusantara" detail="Vendor katering">
          <StatusChip tone="success" className="mt-2">Uang muka lunas</StatusChip>
        </MockItem>
        <MockItem title="Dekorasi Sekar Ayu" detail="Vendor dekorasi">
          <StatusChip tone="success" className="mt-2">Uang muka lunas</StatusChip>
        </MockItem>
        <MockItem title="Gedung Sasana Kriya" detail="Venue">
          <StatusChip tone="success" className="mt-2">Terkonfirmasi</StatusChip>
        </MockItem>
      </>
    ),
  },
  {
    heading: "Termin 2 lunas",
    body: (
      <>
        <MockItem
          title={<span className="tabular-nums">{formatCurrency(20_000_000)}</span>}
          detail="Bukti transfer diunggah 3 Nov, diverifikasi Rina 4 Nov"
        />
        <div className="mt-2.5 flex items-center gap-3 rounded-2xl border border-seal bg-seal-soft px-3.5 py-3">
          <ProofSeal size={44} className="shrink-0 text-seal" />
          <div>
            <p className="font-bold text-seal-text">Terverifikasi</p>
            <p className="text-[12.5px] text-muted-foreground">Klien melihat status yang sama</p>
          </div>
        </div>
      </>
    ),
  },
  {
    heading: "Semua tahapan selesai",
    body: (
      <>
        <MockItem title="24 dari 24 tahapan" detail="Lengkap dengan foto bukti" />
        <MockItem title="Tagihan lunas" detail="3 termin terverifikasi" />
        <MockItem title="Gladi bersih" detail="11 Des 2026, Gedung Sasana Kriya">
          <StatusChip tone="success" className="mt-2">Selesai</StatusChip>
        </MockItem>
      </>
    ),
  },
];

// The date of each step, counted back from the wedding on 12 Des 2026.
const DATES = ["15 Jun 2026", "14 Agu 2026", "10 Nov 2026", "12 Des 2026"];

function StoryScreen({ index }: { index: number }) {
  const screen = SCREENS[index];
  return (
    <>
      <p className="text-[12.5px] font-semibold text-highlight-text">{DATES[index]}</p>
      <p className="mt-1 text-[20px] leading-tight font-bold tracking-[-0.02em]">{screen.heading}</p>
      <div className="mt-2">{screen.body}</div>
    </>
  );
}

function useIsDesktop() {
  const [desktop, setDesktop] = useState(() => window.matchMedia(DESKTOP).matches);
  useEffect(() => {
    const mq = window.matchMedia(DESKTOP);
    const onChange = () => setDesktop(mq.matches);
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, []);
  return desktop;
}

/**
 * "Cara kerja": on a computer the phone stays in view (sticky) and follows the
 * step in the middle of the screen; each step is also a button that shows its
 * screen. Scrolling speed is never changed. On a phone every step carries its
 * own small screen, one under the other.
 */
export default function Story() {
  const desktop = useIsDesktop();
  const [active, setActive] = useState(0);
  const [announcement, setAnnouncement] = useState("");
  const stepRefs = useRef<(HTMLLIElement | null)[]>([]);
  const stageId = useId();

  useEffect(() => {
    if (!desktop || !("IntersectionObserver" in window)) return;
    const io = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) setActive(Number((entry.target as HTMLElement).dataset.step));
        }
      },
      { rootMargin: "-45% 0px -45% 0px" },
    );
    stepRefs.current.forEach((el) => el && io.observe(el));
    return () => io.disconnect();
  }, [desktop]);

  const show = (index: number) => {
    setActive(index);
    setAnnouncement(`Layar menampilkan ${STORY_STEPS[index].when}: ${SCREENS[index].heading}`);
  };

  return (
    <div className="mt-12 lg:mt-16 lg:grid lg:grid-cols-[0.9fr_1.1fr] lg:gap-16">
      {desktop && (
        <div>
          <div className="sticky top-28">
            <figure id={stageId} aria-label="Contoh layar Portal Klien untuk langkah yang dipilih">
              <PhoneFrame className="mx-auto w-[300px]" screenClassName="min-h-[480px]">
                <div key={active} className="motion-safe:animate-in motion-safe:fade-in motion-safe:duration-300">
                  <StoryScreen index={active} />
                </div>
              </PhoneFrame>
            </figure>
            <p className="sr-only" aria-live="polite">
              {announcement}
            </p>
          </div>
        </div>
      )}

      <ol className="grid gap-0">
        {STORY_STEPS.map((step, i) => {
          const on = desktop && active === i;
          return (
            <li
              key={step.when}
              ref={(el) => {
                stepRefs.current[i] = el;
              }}
              data-step={i}
              className={cn(
                "relative border-l-2 py-7 pl-7 lg:flex lg:min-h-[52vh] lg:flex-col lg:justify-center lg:pl-8",
                on ? "border-primary" : "border-border",
              )}
            >
              <span
                aria-hidden
                className={cn(
                  "absolute top-9 -left-[7px] size-3 rounded-full border-2 lg:top-1/2 lg:-mt-1.5",
                  on ? "border-primary bg-primary" : "border-input bg-background",
                )}
              />
              <p className="text-[14px] font-bold text-highlight-text">{step.when}</p>
              <h3 className="mt-1 text-[22px] leading-tight font-bold tracking-[-0.02em] lg:text-[26px]">
                {desktop ? (
                  <button
                    type="button"
                    onClick={() => show(i)}
                    aria-controls={stageId}
                    aria-current={on ? "step" : undefined}
                    className="-mx-2 min-h-11 rounded-md px-2 text-left hover:text-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                  >
                    {step.title}
                  </button>
                ) : (
                  step.title
                )}
              </h3>
              <p className="mt-2 max-w-[440px] text-muted-foreground">{step.body}</p>
              {!desktop && (
                <figure aria-label={`Contoh layar ${step.when}`} className="mt-5 max-w-[340px] rounded-3xl border border-border bg-background p-4 text-[14px] leading-snug">
                  <StoryScreen index={i} />
                </figure>
              )}
            </li>
          );
        })}
      </ol>
    </div>
  );
}
