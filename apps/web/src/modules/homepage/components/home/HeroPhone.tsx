import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { formatCurrency } from "@/shared/lib/formatters";
import { MockItem, PhoneFrame, ProgressRing, StatusChip } from "@/modules/homepage/components/home/mock";

/**
 * The hero's Client Portal phone (example data). Static HTML in the page
 * chunk; the seal is pressed once by the CSS .seal-press moment.
 */
export function HeroPhone() {
  return (
    <figure aria-label="Contoh tampilan Portal Klien dengan data contoh" className="mx-auto w-full max-w-[300px] lg:mr-0 lg:ml-auto">
      <PhoneFrame screenClassName="min-h-[500px]">
        <p className="text-[12.5px] text-muted-foreground">Portal Klien, Raka &amp; Dinda</p>
        <p className="mt-1 text-[21px] leading-tight font-bold tracking-[-0.02em]">32 hari menuju hari H</p>

        <div className="my-4 flex items-center gap-3.5">
          <ProgressRing done={18} total={24} />
          <div>
            <p className="font-bold">18 dari 24 tahapan selesai</p>
            <p className="text-[12.5px] text-muted-foreground">Terakhir: tasting katering</p>
          </div>
        </div>

        <MockItem
          className="relative border-seal bg-seal-soft pr-[84px]"
          title={<>Termin 2 <span className="tabular-nums">{formatCurrency(20_000_000)}</span></>}
          detail={<span className="text-seal-text">Lunas 3 Nov 2026, diverifikasi Rina</span>}
        >
          <ProofSeal size={64} className="seal-press absolute top-1/2 right-2.5 -mt-8 text-seal" />
        </MockItem>
        <MockItem title={<>Pelunasan <span className="tabular-nums">{formatCurrency(10_000_000)}</span></>}>
          <StatusChip tone="warning" className="mt-2">Jatuh tempo 5 Des 2026</StatusChip>
        </MockItem>
        <MockItem title="Fitting baju kedua" detail="2 foto bukti">
          <StatusChip tone="success" className="mt-2">Selesai</StatusChip>
        </MockItem>
      </PhoneFrame>
    </figure>
  );
}
