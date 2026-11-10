import { useState, type ReactNode } from "react";
import { Bookmark, Rows3, Trash2 } from "lucide-react";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Button } from "@/shared/components/kit/button";
import { Input } from "@/shared/components/kit/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/shared/components/kit/popover";
import type { ListDensity } from "@/shared/data-table/useListDensity";
import type { SavedView } from "@/shared/data-table/useSavedViews";

interface Props {
  search: { value: string; onChange: (value: string) => void; placeholder: string };
  /** The list's filters (selects, checkboxes). */
  children?: ReactNode;
  density?: { value: ListDensity; onChange: (d: ListDensity) => void };
  views?: {
    items: SavedView[];
    /** The query a view would save right now (search, filters, sort). */
    current: string;
    onApply: (query: string) => void;
    onSave: (name: string, query: string) => void;
    onRemove: (name: string) => void;
  };
  /** Right side: the list's main action ("Tambah vendor"). */
  actions?: ReactNode;
}

/** Search, filters, density and saved views above a list; the page puts its main action on the right. */
export function ListToolbar({ search, children, density, views, actions }: Props) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <SearchInput className="w-full max-w-xs" placeholder={search.placeholder} value={search.value} onChange={(e) => search.onChange(e.target.value)} aria-label={search.placeholder} />
      {children}
      <div className="ml-auto flex flex-wrap items-center gap-2">
        {views && <SavedViewsMenu {...views} />}
        {density && (
          <Button
            variant="outline"
            size="sm"
            aria-pressed={density.value === "padat"}
            onClick={() => density.onChange(density.value === "padat" ? "normal" : "padat")}
            title="Baris lebih rapat"
          >
            <Rows3 aria-hidden /> {density.value === "padat" ? "Padat" : "Normal"}
          </Button>
        )}
        {actions}
      </div>
    </div>
  );
}

function SavedViewsMenu({ items, current, onApply, onSave, onRemove }: NonNullable<Props["views"]>) {
  const [name, setName] = useState("");
  const [open, setOpen] = useState(false);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={<Button variant="outline" size="sm" />}>
        <Bookmark aria-hidden /> Tampilan
      </PopoverTrigger>
      <PopoverContent align="end" className="w-72">
        <p className="text-[12.5px] font-semibold text-foreground">Tampilan tersimpan</p>
        {items.length === 0 ? (
          <p className="text-[12.5px] text-muted-foreground">Belum ada. Atur pencarian, filter, dan urutan, lalu simpan di bawah. Tampilan hanya tersimpan di peramban ini.</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {items.map((v) => (
              <li key={v.name} className="flex items-center gap-1">
                <Button variant="ghost" size="sm" className="flex-1 justify-start" onClick={() => { onApply(v.query); setOpen(false); }}>
                  {v.name}
                </Button>
                <Button variant="ghost" size="icon-sm" onClick={() => onRemove(v.name)}>
                  <Trash2 aria-hidden />
                  <span className="sr-only">Hapus tampilan {v.name}</span>
                </Button>
              </li>
            ))}
          </ul>
        )}
        <form
          className="flex flex-col gap-2 border-t pt-2.5"
          onSubmit={(e) => {
            e.preventDefault();
            if (!name.trim()) return;
            onSave(name.trim(), current);
            setName("");
          }}
        >
          <label className="text-[12.5px] font-medium text-foreground" htmlFor="view-name">Simpan pengaturan saat ini</label>
          <Input id="view-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="cth. Termin terlambat" maxLength={40} />
          <Button type="submit" size="sm">Simpan tampilan</Button>
        </form>
      </PopoverContent>
    </Popover>
  );
}
