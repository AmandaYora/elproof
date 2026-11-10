import { useId, useRef, useState, type KeyboardEvent } from "react";
import { ChevronDown, EyeOff } from "lucide-react";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { PREVIEW_TABS } from "@/modules/homepage/components/home/previewTabs";
import { PreviewWindow } from "@/modules/homepage/components/home/PreviewWindow";
import { MockItem, PhoneFrame, StatusChip } from "@/modules/homepage/components/home/mock";
import { formatCurrency } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";

const STATS = [
  { value: "18 dari 24", label: "tahapan selesai" },
  { value: formatCurrency(35_000_000), label: `terbayar dari ${formatCurrency(45_000_000)}` },
  { value: "1", label: "kendala perlu ditangani" },
];

const TASKS = [
  { title: "Konfirmasi menu tasting", detail: "Katering Rasa Nusantara, PIC Rina", status: "Besok", tone: "warning" as const },
  { title: "Venue minta jadwal loading dimajukan", detail: "Kendala, dilaporkan 2 hari lalu", status: "Perlu ditangani", tone: "warning" as const },
  { title: "Fitting baju kedua", detail: "Sanggar Kebaya Laras, PIC Yoga", status: "Selesai", tone: "success" as const },
];

const INSTALLMENTS = [
  {
    id: "t1",
    name: "Termin 1, uang muka",
    amount: 15_000_000,
    detail: "Dibayar 2 Agu 2026",
    proof: { uploaded: "2 Agu 2026, 10.15", verified: "Yoga, 3 Agu 2026" },
  },
  {
    id: "t2",
    name: "Termin 2",
    amount: 20_000_000,
    detail: "Dibayar 3 Nov 2026",
    proof: { uploaded: "3 Nov 2026, 19.42", verified: "Rina, 4 Nov 2026" },
  },
  { id: "t3", name: "Pelunasan", amount: 10_000_000, detail: "Jatuh tempo 5 Des 2026", proof: null },
];

function ProjectPanel() {
  return (
    <div className="p-5 sm:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-[20px] font-bold tracking-[-0.02em]">Raka &amp; Dinda</p>
          <p className="mt-1 text-[14px] text-muted-foreground">Sabtu, 12 Desember 2026, Gedung Sasana Kriya</p>
        </div>
        <span className="rounded-full bg-accent px-3 py-1 text-[13px] font-bold text-accent-foreground">H-32</span>
      </div>
      <dl className="mt-5 grid gap-2.5 sm:grid-cols-3">
        {STATS.map((stat) => (
          <div key={stat.label} className="flex flex-col-reverse rounded-xl border border-border px-4 py-3">
            <dt className="text-[13px] text-muted-foreground">{stat.label}</dt>
            <dd className="text-[19px] font-bold tracking-[-0.02em] tabular-nums">{stat.value}</dd>
          </div>
        ))}
      </dl>
      <p className="mt-6 text-[13px] font-semibold text-muted-foreground">Yang perlu dikerjakan</p>
      <ul className="mt-2 divide-y divide-border border-y border-border">
        {TASKS.map((task) => (
          <li key={task.title} className="flex items-center justify-between gap-3 py-3">
            <div className="min-w-0">
              <p className="text-[14.5px] font-semibold">{task.title}</p>
              <p className="text-[13px] text-muted-foreground">{task.detail}</p>
            </div>
            <StatusChip tone={task.tone}>{task.status}</StatusChip>
          </li>
        ))}
      </ul>
    </div>
  );
}

function InstallmentPanel() {
  const [open, setOpen] = useState<string | null>(null);
  const baseId = useId();
  return (
    <div className="p-5 sm:p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-[20px] font-bold tracking-[-0.02em]">Tagihan Raka &amp; Dinda</p>
          <p className="mt-1 text-[14px] text-muted-foreground">3 termin, sisa {formatCurrency(10_000_000)}</p>
        </div>
        <StatusChip tone="success">Sesuai jadwal</StatusChip>
      </div>
      <ul className="mt-5 divide-y divide-border border-y border-border">
        {INSTALLMENTS.map((item) => {
          const expanded = open === item.id;
          const detailId = `${baseId}-${item.id}`;
          return (
            <li key={item.id} className="py-3">
              <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
                <div className="min-w-0 flex-1">
                  <p className="text-[14.5px] font-semibold">{item.name}</p>
                  <p className="text-[13px] text-muted-foreground">{item.detail}</p>
                </div>
                <p className="text-[15px] font-bold tabular-nums">{formatCurrency(item.amount)}</p>
                {item.proof ? (
                  <button
                    type="button"
                    aria-expanded={expanded}
                    aria-controls={detailId}
                    onClick={() => setOpen(expanded ? null : item.id)}
                    className="inline-flex min-h-11 items-center gap-1.5 rounded-full border border-seal bg-seal-soft px-3.5 text-[13.5px] font-semibold text-seal-text hover:bg-background focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                  >
                    {expanded ? "Tutup bukti" : "Lihat bukti"}
                    <ChevronDown aria-hidden className={cn("size-4 motion-safe:transition-transform", expanded && "rotate-180")} />
                  </button>
                ) : (
                  <StatusChip tone="warning" className="min-h-8">Belum dibayar</StatusChip>
                )}
              </div>
              {item.proof && (
                <div id={detailId} hidden={!expanded} className="mt-3 rounded-xl border border-border bg-muted p-3.5">
                  <div className="flex gap-3.5">
                    <div aria-hidden className="grid h-[88px] w-[66px] shrink-0 content-start gap-1.5 rounded-md border border-border bg-background p-2">
                      <span className="h-1.5 w-8 rounded-full bg-accent" />
                      <span className="h-1.5 w-11 rounded-full bg-border" />
                      <span className="h-1.5 w-9 rounded-full bg-border" />
                      <span className="mt-2 h-2 w-10 rounded-full bg-primary/70" />
                    </div>
                    <div className="min-w-0 text-[13.5px]">
                      <p className="font-bold">Bukti transfer {formatCurrency(item.amount)}</p>
                      <p className="mt-1 text-muted-foreground">Diunggah klien {item.proof.uploaded}</p>
                      <p className="text-muted-foreground">Diverifikasi {item.proof.verified}</p>
                      <p className="mt-2 flex items-center gap-2 font-semibold text-seal-text">
                        <ProofSeal size={28} className="shrink-0 text-seal" />
                        Tampil sama di Portal Klien
                      </p>
                    </div>
                  </div>
                </div>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

function PortalPanel() {
  return (
    <div className="grid gap-6 p-5 sm:p-6 md:grid-cols-[1fr_auto] md:items-start">
      <div className="max-w-[46ch]">
        <p className="text-[20px] font-bold tracking-[-0.02em]">Yang dilihat klien</p>
        <p className="mt-2 text-[15px] text-muted-foreground">
          Klien membuka Portal Klien dari tautan WhatsApp, tanpa memasang aplikasi. Mereka melihat jadwal, tagihan,
          dan bukti yang Anda pilih untuk ditampilkan.
        </p>
        <p className="mt-4 flex items-start gap-2 text-[14px] font-semibold">
          <EyeOff aria-hidden className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
          Biaya vendor dan catatan internal tim tidak pernah tampil di sini.
        </p>
      </div>
      <PhoneFrame className="mx-auto w-[236px] p-2.5" screenClassName="min-h-[300px] px-3.5 py-4 text-[13px]">
        <p className="text-[12px] text-muted-foreground">Halo, Dinda</p>
        <p className="mt-0.5 text-[16px] leading-tight font-bold">32 hari lagi menuju hari H</p>
        <div className="mt-3 h-2 overflow-hidden rounded-full bg-accent" aria-hidden>
          <div className="h-full w-3/4 rounded-full bg-primary" />
        </div>
        <p className="mt-1.5 text-[12px] text-muted-foreground">18 dari 24 tahapan selesai</p>
        <MockItem title={<>Pelunasan <span className="tabular-nums">{formatCurrency(10_000_000)}</span></>} detail="Jatuh tempo 5 Des 2026" className="px-3 py-2.5" />
        <MockItem title="Termin 2 lunas" className="border-seal bg-seal-soft px-3 py-2.5">
          <p className="mt-0.5 flex items-center gap-1.5 text-[12px] font-semibold text-seal-text">
            <ProofSeal size={22} className="shrink-0 text-seal" />
            Terverifikasi
          </p>
        </MockItem>
      </PhoneFrame>
    </div>
  );
}

const PANELS = [ProjectPanel, InstallmentPanel, PortalPanel];

/**
 * The try-it preview: three tabs of a small WO Console with example data.
 * APG tabs: arrows move and select, Home/End jump, one tab stop.
 */
export default function AppPreview() {
  const [selected, setSelected] = useState(0);
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const baseId = useId();

  const select = (index: number) => {
    const next = (index + PREVIEW_TABS.length) % PREVIEW_TABS.length;
    setSelected(next);
    tabRefs.current[next]?.focus();
  };

  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    const moves: Record<string, number> = { ArrowRight: selected + 1, ArrowLeft: selected - 1, Home: 0, End: PREVIEW_TABS.length - 1 };
    if (!(event.key in moves)) return;
    event.preventDefault();
    select(moves[event.key]);
  };

  return (
    <PreviewWindow
      tabs={
        <div role="tablist" aria-label="Bagian pratinjau" className="flex gap-1 overflow-x-auto py-2 [scrollbar-width:none]">
          {PREVIEW_TABS.map((tab, i) => {
            const on = i === selected;
            return (
              <button
                key={tab.id}
                ref={(el) => {
                  tabRefs.current[i] = el;
                }}
                type="button"
                role="tab"
                id={`${baseId}-tab-${tab.id}`}
                aria-selected={on}
                aria-controls={`${baseId}-panel-${tab.id}`}
                tabIndex={on ? 0 : -1}
                onClick={() => setSelected(i)}
                onKeyDown={onKeyDown}
                className={cn(
                  "inline-flex min-h-11 shrink-0 items-center rounded-lg border px-3.5 text-[14px] font-semibold whitespace-nowrap focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-ring",
                  on ? "border-border bg-background text-foreground" : "border-transparent text-muted-foreground hover:text-foreground",
                )}
              >
                {tab.label}
              </button>
            );
          })}
        </div>
      }
    >
      {PREVIEW_TABS.map((tab, i) => {
        const Panel = PANELS[i];
        return (
          <div
            key={tab.id}
            role="tabpanel"
            id={`${baseId}-panel-${tab.id}`}
            aria-labelledby={`${baseId}-tab-${tab.id}`}
            tabIndex={0}
            hidden={i !== selected}
            className="focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-ring"
          >
            <Panel />
          </div>
        );
      })}
    </PreviewWindow>
  );
}
