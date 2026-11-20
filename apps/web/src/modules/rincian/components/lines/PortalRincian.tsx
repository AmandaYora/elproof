import { useState } from "react";
import { formatCurrency } from "@/shared/lib/formatters";
import { extraGuestsPlain, itemFormula } from "@/modules/rincian/lib/describe";
import { negative } from "@/modules/rincian/components/lines/CostSummaryPanel";
import type { Rincian, RincianLine } from "@/modules/rincian/types";

/** How many isi of a package show before "Lihat n isi lainnya". */
const FIRST_COMPONENTS = 3;

/**
 * "Paket & rincian" for the couple (DESAIN §6.11): line names and the isi of each package. Amounts, PPN and the
 * contract value only when the server sent them (portal.finance.view); an isi never has a price.
 */
export function PortalRincian({ rincian }: { rincian: Rincian }) {
  const withMoney = rincian.totals !== undefined;
  const lines = rincian.lines;
  const componentsOf = (pkg: RincianLine) => lines.filter((l) => l.kind === "Component" && l.parentId === pkg.id);
  const extraOf = (pkg: RincianLine) => lines.find((l) => l.kind === "ExtraGuests" && l.parentId === pkg.id);
  const main = lines.filter((l) => l.kind === "Package" || (l.kind === "Item" && !l.isBonus));
  const bonus = lines.filter((l) => l.kind === "Item" && l.isBonus);
  const discounts = lines.filter((l) => l.kind === "Discount");

  return (
    <div className="flex flex-col gap-4">
      {withMoney && rincian.totals && (
        <div>
          <p className="text-[13px] text-muted-foreground">Nilai kontrak</p>
          <p className="text-[24px] leading-tight font-semibold tabular-nums text-foreground">{formatCurrency(rincian.totals.total)}</p>
          {rincian.isPkp && rincian.totals.tax > 0 && (
            <p className="text-[12.5px] text-muted-foreground">termasuk PPN {formatCurrency(rincian.totals.tax)}</p>
          )}
        </div>
      )}
      <ul className="flex flex-col divide-y divide-border-light border-t border-border-light">
        {main.map((l) =>
          l.kind === "Package" ? (
            <PackageItem key={l.id} line={l} components={componentsOf(l)} extra={extraOf(l)} withMoney={withMoney} />
          ) : (
            <li key={l.id} className="py-3">
              <Line name={l.description} amount={withMoney ? l.amount : undefined} />
              <p className="mt-0.5 text-[12.5px] text-muted-foreground">{itemFormula(l.quantity, l.unit, withMoney ? l.unitPrice : undefined)}</p>
            </li>
          ),
        )}
        {bonus.length > 0 && (
          <li className="py-3 text-[13.5px] text-foreground">
            <span className="font-medium">Bonus:</span> {bonus.map((b) => b.description).join(", ")}
          </li>
        )}
        {discounts.map((d) => (
          <li key={d.id} className="py-3">
            <Line name={d.description || "Potongan"} amount={withMoney && d.amount !== undefined ? d.amount : undefined} negativeAmount />
          </li>
        ))}
      </ul>
    </div>
  );
}

function PackageItem({ line, components, extra, withMoney }: { line: RincianLine; components: RincianLine[]; extra?: RincianLine; withMoney: boolean }) {
  const [all, setAll] = useState(false);
  const shown = all ? components : components.slice(0, FIRST_COMPONENTS);
  const rest = components.length - shown.length;
  return (
    <li className="py-3">
      <Line name={line.description} amount={withMoney ? line.amount : undefined} strong />
      {components.length > 0 && (
        <ul className="mt-1.5 list-disc pl-5 text-[13px] text-foreground marker:text-muted-foreground">
          {shown.map((c) => (
            <li key={c.id}>{c.description}</li>
          ))}
        </ul>
      )}
      {rest > 0 && (
        <button type="button" onClick={() => setAll(true)} className="mt-1 text-[13px] font-medium text-primary hover:underline">
          Lihat {rest} isi lainnya
        </button>
      )}
      {extra && extra.extraGuests && !extra.extraGuests.noGuests && extra.extraGuests.over > 0 && (
        <div className="mt-2">
          <Line name="Tambahan tamu" amount={withMoney ? extra.amount : undefined} />
          <p className="mt-0.5 text-[12.5px] text-muted-foreground">{extraGuestsPlain(extra.extraGuests)}</p>
        </div>
      )}
    </li>
  );
}

function Line({ name, amount, strong = false, negativeAmount = false }: { name: string; amount?: number; strong?: boolean; negativeAmount?: boolean }) {
  return (
    <p className="flex items-baseline justify-between gap-3 text-[13.5px]">
      <span className={strong ? "font-semibold text-foreground" : "text-foreground"}>{name}</span>
      {amount !== undefined && (
        <span className="whitespace-nowrap font-medium tabular-nums text-foreground">{negativeAmount ? negative(amount) : formatCurrency(amount)}</span>
      )}
    </p>
  );
}
