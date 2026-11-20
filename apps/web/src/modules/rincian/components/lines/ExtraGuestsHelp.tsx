import { Info } from "lucide-react";
import { Popover, PopoverContent, PopoverTrigger } from "@/shared/components/kit/popover";
import { formatCurrency } from "@/shared/lib/formatters";
import type { ExtraGuestsInfo } from "@/modules/rincian/types";

/** Toggletip "Cara menghitung" on a "Tambahan tamu" row (DESAIN §5.3): which acara decides it, the rule, and the way out. */
export function ExtraGuestsHelp({ info, packageName }: { info: ExtraGuestsInfo; packageName: string }) {
  const rule =
    info.step > 0
      ? `${info.includedGuests.toLocaleString("id-ID")} tamu termasuk, tambahan per ${info.step.toLocaleString("id-ID")} tamu${
          info.stepPrice !== undefined ? ` ${formatCurrency(info.stepPrice)}` : ""
        }`
      : `${info.includedGuests.toLocaleString("id-ID")} tamu termasuk, tanpa harga tambahan`;
  return (
    <Popover>
      <PopoverTrigger
        aria-label={`Cara menghitung tambahan tamu ${packageName}`}
        className="inline-flex size-6 items-center justify-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
      >
        <Info aria-hidden className="size-3.5" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-80 text-[13px]">
        <p className="font-semibold text-foreground">Cara menghitung</p>
        <p className="text-foreground">
          {info.noGuests
            ? `${info.eventKindName || "Acara paket ini"} belum punya tamu, jadi tambahan tamu belum dihitung.`
            : `${info.eventKindName}, ${info.guests.toLocaleString("id-ID")} tamu: acara terbesar yang dicakup paket.`}
        </p>
        <p className="text-muted-foreground">Aturan paket: {rule}.</p>
        <p className="text-muted-foreground">Mau harga lain? Tambahkan Potongan.</p>
      </PopoverContent>
    </Popover>
  );
}
