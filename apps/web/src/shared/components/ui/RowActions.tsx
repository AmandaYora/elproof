import type { ReactNode } from "react";
import { Button } from "@/shared/components/kit/button";
import { ActionMenu, type ActionMenuItem } from "@/shared/components/ui/ActionMenu";

interface RowActionsProps {
  /** The one action people use most on this row (usually "Ubah"), kept visible as an icon button. */
  primary?: { label: string; icon: ReactNode; onClick: () => void; hidden?: boolean };
  /** Everything else (nonaktifkan, hapus, …) behind "⋯". */
  items?: ActionMenuItem[];
  /** Names the row for assistive technology ("Katering Rasa Nusantara"). */
  rowLabel: string;
}

/**
 * Row actions in a table or a phone card (RISET WO Console §3 #7): at most one visible icon, the rest in a "⋯" menu, so
 * rows stay calm and every row has its actions in the same place. Opening the record is the row's own link.
 */
export function RowActions({ primary, items = [], rowLabel }: RowActionsProps) {
  return (
    <div className="flex items-center justify-end gap-0.5">
      {primary && !primary.hidden && (
        <Button variant="ghost" size="icon-sm" onClick={primary.onClick} aria-label={`${primary.label} ${rowLabel}`} title={primary.label} className="text-muted-foreground">
          {primary.icon}
        </Button>
      )}
      <ActionMenu variant="ghost" items={items} label={`Aksi lain untuk ${rowLabel}`} />
    </div>
  );
}
