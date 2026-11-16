import { useState, type ReactNode } from "react";
import { SlidersHorizontal } from "lucide-react";
import { Button } from "@/shared/components/kit/button";
import { Drawer } from "@/shared/components/ui/Drawer";

interface Props {
  /** How many filters are set right now: shown on the phone button ("Saring (2)"). */
  count: number;
  /** Clears the filters (not the search). */
  onClear: () => void;
  /** The list's filters (selects, comboboxes). */
  children: ReactNode;
}

/**
 * A list's filters (RISET WO Console §3 #8): inline in the toolbar on a computer; on a phone one "Saring (n)" button that
 * opens them in a drawer, so the filters never fill the first screen before the list starts.
 */
export function ListFilters({ count, onClear, children }: Props) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <div className="hidden md:contents">{children}</div>
      <Button variant="outline" className="h-9 md:hidden" aria-haspopup="dialog" onClick={() => setOpen(true)}>
        <SlidersHorizontal aria-hidden /> Saring{count > 0 ? ` (${count})` : ""}
      </Button>
      {open && (
        <Drawer
          open
          onClose={() => setOpen(false)}
          title="Saring daftar"
          footer={
            <>
              {count > 0 && (
                <Button variant="ghost" onClick={onClear}>
                  Hapus saringan
                </Button>
              )}
              <Button onClick={() => setOpen(false)}>Tampilkan hasil</Button>
            </>
          }
        >
          <div className="flex flex-col gap-3 *:w-full!">{children}</div>
        </Drawer>
      )}
    </>
  );
}

/** The number of filters that have a value. */
export function activeFilterCount(filters: Record<string, string>): number {
  return Object.values(filters).filter((v) => v !== "").length;
}
