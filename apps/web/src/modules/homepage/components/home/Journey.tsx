import type { ReactNode } from "react";
import { CalendarDays, EyeOff, FileImage, MessageSquareWarning } from "lucide-react";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { StatusChip } from "@/modules/homepage/components/home/mock";
import { formatCurrency } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";

/*
 * "Cara kerja" on the Beranda: one wedding, H-180 to the day itself, as five
 * cards of different sizes. The H-markers are real information (the order
 * work happens in), and every card shows a small piece of the actual screen
 * instead of an icon. Example data; plain HTML (prerendered, no chunk).
 */

function Card({ when, title, body, children, className, tone = "plain" }: {
  when: string;
  title: string;
  body: string;
  children: ReactNode;
  className?: string;
  tone?: "plain" | "seal";
}) {
  return (
    <li
      className={cn(
        "reveal flex flex-col rounded-[20px] border p-5 sm:p-6",
        tone === "seal" ? "border-seal/60 bg-seal-soft" : "border-border bg-card",
        className,
      )}
    >
      <p className={cn("text-[14px] font-bold tabular-nums", tone === "seal" ? "text-seal-text" : "text-highlight-text")}>{when}</p>
      <h3 className="mt-1.5 text-[20px] leading-snug font-bold tracking-[-0.02em]">{title}</h3>
      <p className="mt-2 max-w-[44ch] text-[15px] text-muted-foreground">{body}</p>
      <div aria-hidden className="mt-5 flex-1 content-end">
        {children}
      </div>
    </li>
  );
}

/** The light panel each card's screen piece sits in. */
function Screen({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("rounded-xl border border-border bg-background p-3 text-[13px] leading-snug", className)}>{children}</div>;
}

function Row({ left, right, sub }: { left: ReactNode; right: ReactNode; sub?: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 py-2 first:pt-0 last:pb-0">
      <div className="min-w-0">
        <p className="truncate font-semibold">{left}</p>
        {sub && <p className="truncate text-[12px] text-muted-foreground">{sub}</p>}
      </div>
      <div className="shrink-0">{right}</div>
    </div>
  );
}

export function Journey() {
  return (
    <ol className="mt-12 grid gap-4 md:grid-cols-6 md:gap-5">
      <Card
        className="md:col-span-3"
        when="H-180"
        title="Kontrak dan termin dibuat sekali"
        body="Isi data pasangan, paket, dan nilai kontrak. Termin terhitung dari templat Anda dan bisa dihitung ulang bila nilai kontrak berubah."
      >
        <Screen className="divide-y divide-border">
          <Row left="Uang muka 30%" sub="15 Jun 2026" right={<span className="font-semibold tabular-nums">{formatCurrency(13_500_000)}</span>} />
          <Row left="Termin 2, 40%" sub="1 Nov 2026" right={<span className="font-semibold tabular-nums">{formatCurrency(18_000_000)}</span>} />
          <Row left="Pelunasan 30%" sub="5 Des 2026" right={<span className="font-semibold tabular-nums">{formatCurrency(13_500_000)}</span>} />
        </Screen>
      </Card>

      <Card
        className="md:col-span-3"
        when="H-150"
        title="Jadwal persiapan mengikuti hari H"
        body="Tahapan dari jadwal bawaan Anda diberi tanggal mundur dari hari acara. Tanggal acara bergeser, jadwalnya ikut bergeser."
      >
        <Screen className="divide-y divide-border">
          <Row left="Survei venue dan vendor" right={<span className="text-[12px] font-semibold text-muted-foreground tabular-nums">H-90</span>} />
          <Row left="Technical meeting" right={<span className="text-[12px] font-semibold text-muted-foreground tabular-nums">H-14</span>} />
          <Row
            left={
              <span className="inline-flex items-center gap-1.5">
                <CalendarDays className="size-3.5 text-muted-foreground" /> Hari pernikahan
              </span>
            }
            right={<span className="text-[12px] font-semibold text-foreground">12 Des 2026</span>}
          />
        </Screen>
      </Card>

      <Card
        className="md:col-span-2"
        when="H-120"
        title="Vendor dan venue per proyek"
        body="Kontrak, uang muka, dan sisa pembayaran vendor tersimpan di proyeknya."
      >
        <Screen className="divide-y divide-border">
          <Row left="Katering Rasa Nusantara" right={<StatusChip tone="success">Uang muka lunas</StatusChip>} />
          <Row left="Dekorasi Sekar Ayu" right={<StatusChip tone="neutral">Negosiasi</StatusChip>} />
        </Screen>
        <p className="mt-3 flex items-center gap-1.5 text-[13px] text-muted-foreground">
          <EyeOff className="size-3.5" /> Biaya vendor tidak pernah tampil ke klien
        </p>
      </Card>

      <Card
        className="md:col-span-2"
        tone="seal"
        when="H-32"
        title="Setiap pembayaran ada buktinya"
        body="Tim mencatat pembayaran beserta bukti transfer, lalu memverifikasinya. Segel ini hanya muncul pada yang sudah diperiksa."
      >
        <Screen className="flex items-center gap-3 border-seal/40">
          <span className="grid size-11 shrink-0 place-items-center rounded-lg bg-muted text-muted-foreground">
            <FileImage className="size-5" />
          </span>
          <div className="min-w-0 flex-1">
            <p className="font-semibold">Termin 2</p>
            <p className="text-[12px] text-muted-foreground tabular-nums">{formatCurrency(20_000_000)}, bukti_transfer.jpg</p>
          </div>
          <ProofSeal size={40} className="shrink-0 text-seal" />
        </Screen>
      </Card>

      <Card
        className="md:col-span-2"
        when="H-7"
        title="Kendala ditangani, klien ikut tahu"
        body="Catat kendala per vendor atau untuk acara secara umum, lalu pilih apakah pasangan perlu melihatnya."
      >
        <Screen>
          <div className="flex items-start gap-2.5">
            <MessageSquareWarning className="mt-0.5 size-4 shrink-0 text-warning" />
            <div className="min-w-0">
              <p className="font-semibold">Venue minta jadwal loading dimajukan</p>
              <p className="mt-0.5 text-[12px] text-muted-foreground">Dilaporkan 2 hari lalu, terlihat oleh pasangan</p>
            </div>
          </div>
          <StatusChip tone="success" className="mt-2.5">Selesai</StatusChip>
        </Screen>
      </Card>
    </ol>
  );
}
