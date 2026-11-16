import { CircleAlert, Clock3 } from "lucide-react";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { MockItem, PhoneFrame, ProgressRing, StatusChip } from "@/modules/homepage/components/home/mock";
import { formatCurrency } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";

/*
 * The hero's product picture (example data): the WO Console the team works in,
 * and in front of it the Client Portal the couple opens from WhatsApp. One
 * payment, "Termin 2", is verified in the Console and carries the same gold seal
 * on the phone: the brand's promise drawn once. Plain HTML in the page chunk, so
 * it is in the prerendered page; the seal is pressed once by CSS (.seal-press).
 */

const MENU = ["Beranda", "Proyek", "Klien", "Vendor", "Venue"];

const PAYMENTS = [
  {
    label: "Uang muka",
    amount: 15_000_000,
    state: "done" as const,
    note: "Diverifikasi Rina, 2 Agu",
  },
  {
    label: "Termin 2",
    amount: 20_000_000,
    state: "sealed" as const,
    note: "Diverifikasi Rina, 4 Nov",
  },
  {
    label: "Pelunasan",
    amount: 10_000_000,
    state: "due" as const,
    note: "Jatuh tempo 5 Des",
  },
];

const UPCOMING = [
  {
    name: "Konfirmasi menu tasting",
    who: "Katering Rasa Nusantara",
    when: "Besok",
  },
  { name: "Fitting baju kedua", who: "Sanggar Kebaya Laras", when: "H-21" },
  { name: "Technical meeting", who: "Semua vendor", when: "H-14" },
];

function ConsoleWindow() {
  return (
    <div className="overflow-hidden rounded-[14px] border border-border bg-card text-[13px] leading-snug text-foreground shadow-[0_30px_60px_-30px] shadow-primary/40">
      <div className="flex h-10 items-center justify-between border-b border-border bg-muted px-4">
        <p className="text-[12.5px] font-bold">WO Console</p>
        <p className="text-[12px] text-muted-foreground">Proyek / Raka &amp; Dinda</p>
      </div>
      <div className="grid grid-cols-[132px_1fr]">
        <div className="border-r border-border px-2.5 py-3">
          {MENU.map((item, i) => (
            <span
              key={item}
              className={cn(
                "block rounded-md px-2.5 py-1.5 text-[12.5px]",
                i === 1 ? "bg-accent font-semibold text-accent-foreground" : "text-muted-foreground",
              )}
            >
              {item}
            </span>
          ))}
        </div>
        <div className="min-w-0 p-4">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-[17px] font-bold tracking-[-0.02em]">Raka &amp; Dinda</p>
              <p className="mt-0.5 text-[12px] text-muted-foreground">Sabtu, 12 Desember 2026, Gedung Sasana Kriya</p>
            </div>
            <span className="rounded-full bg-accent px-2.5 py-1 text-[12px] font-semibold text-accent-foreground tabular-nums">
              H-32
            </span>
          </div>

          <div className="mt-3.5 grid grid-cols-3 gap-2.5">
            <div className="rounded-lg border border-border p-2.5">
              <p className="text-[11.5px] text-muted-foreground">Tahapan</p>
              <p className="mt-0.5 font-bold tabular-nums">18 dari 24</p>
              <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-muted">
                <div className="h-full w-3/4 rounded-full bg-primary" />
              </div>
            </div>
            <div className="rounded-lg border border-border p-2.5">
              <p className="text-[11.5px] text-muted-foreground">Sudah dibayar</p>
              <p className="mt-0.5 font-bold tabular-nums">{formatCurrency(35_000_000)}</p>
              <p className="text-[11.5px] text-muted-foreground tabular-nums">dari {formatCurrency(45_000_000)}</p>
            </div>
            <div className="rounded-lg border border-border p-2.5">
              <p className="text-[11.5px] text-muted-foreground">Kendala terbuka</p>
              <p className="mt-0.5 flex items-center gap-1 font-bold">
                <CircleAlert aria-hidden className="size-3.5 text-warning" /> 1
              </p>
              <p className="text-[11.5px] text-muted-foreground">Jadwal loading venue</p>
            </div>
          </div>

          <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]">
            <div>
              <p className="text-[12.5px] font-semibold">Pembayaran klien</p>
              <ul className="mt-1.5 divide-y divide-border rounded-lg border border-border">
                {PAYMENTS.map((p) => (
                  <li
                    key={p.label}
                    className={cn("flex h-[50px] items-center gap-3 px-3", p.state === "sealed" && "bg-seal-soft")}
                  >
                    <div className="min-w-0 flex-1">
                      <p className="font-semibold">{p.label}</p>
                      <p className="truncate text-[11.5px] text-muted-foreground">{p.note}</p>
                    </div>
                    <p className="font-semibold tabular-nums">{formatCurrency(p.amount)}</p>
                    <span className="flex w-[124px] shrink-0 justify-end whitespace-nowrap">
                      {p.state === "sealed" ? (
                        <span className="inline-flex items-center gap-1.5 text-[12px] font-semibold text-seal-text">
                          <ProofSeal size={26} className="text-seal" /> Terverifikasi
                        </span>
                      ) : p.state === "done" ? (
                        <StatusChip tone="success">Terverifikasi</StatusChip>
                      ) : (
                        <span className="inline-flex items-center gap-1 rounded-full bg-warning-soft px-2.5 py-1 text-[12px] font-semibold text-warning-text">
                          <Clock3 aria-hidden className="size-3.5" /> Belum dibayar
                        </span>
                      )}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
            <div className="hidden lg:block">
              <p className="text-[12.5px] font-semibold">Tahapan berikutnya</p>
              <ul className="mt-1.5 divide-y divide-border rounded-lg border border-border">
                {UPCOMING.map((m) => (
                  <li key={m.name} className="flex h-[50px] items-center justify-between gap-3 px-3">
                    <div className="min-w-0">
                      <p className="truncate font-semibold">{m.name}</p>
                      <p className="truncate text-[11.5px] text-muted-foreground">{m.who}</p>
                    </div>
                    <span className="shrink-0 text-[12px] font-semibold text-muted-foreground tabular-nums">{m.when}</span>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function PortalPhone() {
  return (
    <PhoneFrame screenClassName="px-4 py-5">
      <p className="text-[12px] text-muted-foreground">Portal Klien, Raka &amp; Dinda</p>
      <p className="mt-1 text-[19px] leading-tight font-bold tracking-[-0.02em]">32 hari menuju hari H</p>
      <div className="my-3.5 flex items-center gap-3">
        <ProgressRing done={18} total={24} size={52} />
        <div>
          <p className="text-[13.5px] font-bold">18 dari 24 tahapan selesai</p>
          <p className="text-[12px] text-muted-foreground">Terakhir: tasting katering</p>
        </div>
      </div>
      <MockItem
        className="relative border-seal bg-seal-soft pr-[72px]"
        title={
          <>
            Termin 2 <span className="tabular-nums">{formatCurrency(20_000_000)}</span>
          </>
        }
        detail={<span className="text-seal-text">Lunas, diverifikasi Rina 4 Nov</span>}
      >
        <ProofSeal size={56} className="seal-press absolute top-1/2 right-2 -mt-7 text-seal" />
      </MockItem>
      <MockItem
        title={
          <>
            Pelunasan <span className="tabular-nums">{formatCurrency(10_000_000)}</span>
          </>
        }
      >
        <StatusChip tone="warning" className="mt-2">
          Jatuh tempo 5 Des 2026
        </StatusChip>
      </MockItem>
    </PhoneFrame>
  );
}

/** The two screens, one record. Decorative for screen readers beyond the figure's own label. */
export function HeroScreens() {
  const caption = (
    <>
      Tim memverifikasi Termin 2 di WO Console. Pasangan langsung melihat segel yang sama di Portal Klien.{" "}
      <span className="text-subtle-foreground">Contoh data.</span>
    </>
  );
  return (
    <figure className="relative" aria-label="Contoh tampilan WO Console dan Portal Klien dengan data contoh">
      <div className="relative sm:min-h-[560px] lg:min-h-[620px]">
        <div className="hidden sm:ml-[16%] sm:block lg:ml-[19%]">
          <div aria-hidden>
            <ConsoleWindow />
          </div>
          <p className="mt-5 max-w-[52ch] pl-[30%] text-[14px] text-muted-foreground lg:pl-[24%]">{caption}</p>
        </div>
        <div aria-hidden className="mx-auto w-full max-w-[264px] sm:absolute sm:top-[170px] sm:left-0 sm:mx-0 sm:w-[236px] lg:top-[200px] lg:w-[256px]">
          <PortalPhone />
        </div>
      </div>
      <p className="mx-auto mt-5 max-w-[48ch] text-center text-[14px] text-muted-foreground sm:hidden">{caption}</p>
    </figure>
  );
}
