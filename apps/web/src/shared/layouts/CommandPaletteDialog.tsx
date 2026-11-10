import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { FolderKanban, Store, Users, type LucideIcon } from "lucide-react";
import { Command, CommandDialog, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/shared/components/kit/command";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { NavItem } from "@/shared/layouts/nav/types";

interface Hit {
  key: string;
  label: string;
  sublabel: string;
  icon: LucideIcon;
  path: string;
}

const PER_KIND = 4;
const MIN_CHARS = 2;

interface Named {
  id: number | string;
  name: string;
  status?: string;
}

/** One kind of result: a small page searched on the server, never the whole list. */
async function searchKind(url: string, q: string, signal: AbortSignal): Promise<Named[]> {
  const res = await httpClient.get(url, { params: { search: q, limit: PER_KIND }, signal });
  return res.data.data as Named[];
}

const settle = <T,>(p: Promise<T>) => p.then((value) => ({ ok: true as const, value }), () => ({ ok: false as const }));


/** The search box and its results. Loaded the first time the palette is opened (cmdk and its dialog are not needed to show a page). */
export default function CommandPaletteDialog({ open, onOpenChange: setOpen, pages, searchRecords }: { open: boolean; onOpenChange: (open: boolean) => void; pages: NavItem[]; searchRecords: boolean }) {
  const [query, setQuery] = useState("");
  const [hits, setHits] = useState<Hit[]>([]);
  const [searching, setSearching] = useState(false);
  const navigate = useNavigate();
  const q = query.trim();
  const debounced = useDebouncedValue(q, 250);

  useEffect(() => {
    if (!open) {
      setQuery("");
      setHits([]);
    }
  }, [open]);

  useEffect(() => {
    if (!searchRecords || debounced.length < MIN_CHARS) {
      setHits([]);
      setSearching(false);
      return;
    }
    const controller = new AbortController();
    setSearching(true);
    Promise.all([
      settle(searchKind(API.projects.base, debounced, controller.signal)),
      settle(searchKind(API.vendors.base, debounced, controller.signal)),
      settle(searchKind(API.clients.base, debounced, controller.signal)),
    ]).then(([projects, vendors, clients]) => {
      if (controller.signal.aborted) return;
      setHits([
        ...(projects.ok ? projects.value : []).map((p) => ({ key: `p-${p.id}`, label: p.name, sublabel: "Proyek", icon: FolderKanban, path: ROUTE_PATHS.projectDetail(String(p.id)) })),
        ...(vendors.ok ? vendors.value : []).map((v) => ({ key: `v-${v.id}`, label: v.name, sublabel: "Vendor", icon: Store, path: `${ROUTE_PATHS.vendors}?q=${encodeURIComponent(v.name)}` })),
        ...(clients.ok ? clients.value : []).map((c) => ({ key: `c-${c.id}`, label: c.name, sublabel: "Klien", icon: Users, path: `${ROUTE_PATHS.clients}?q=${encodeURIComponent(c.name)}` })),
      ].slice(0, 9));
      setSearching(false);
    });
    return () => controller.abort();
  }, [debounced, searchRecords]);

  const matchingPages = pages.filter((p) => p.label.toLowerCase().includes(q.toLowerCase()));

  function go(path: string) {
    setOpen(false);
    navigate(path);
  }

  const pending = searching || q !== debounced;
  return (
    <>
      <CommandDialog open={open} onOpenChange={setOpen} title="Pencarian" description={searchRecords ? "Cari proyek, vendor, klien, atau halaman, lalu tekan Enter." : "Cari halaman, lalu tekan Enter."}>
        <Command shouldFilter={false}>
          <CommandInput value={query} onValueChange={setQuery} placeholder={searchRecords ? "Cari proyek, vendor, klien, atau halaman..." : "Cari halaman..."} />
          <CommandList>
            {matchingPages.length > 0 && (
              <CommandGroup heading="Halaman">
                {matchingPages.map((p) => (
                  <CommandItem key={p.to} value={`page-${p.to}`} onSelect={() => go(p.to)}>
                    <p.icon aria-hidden />
                    {p.label}
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
            {hits.length > 0 && (
              <CommandGroup heading="Hasil pencarian">
                {hits.map((h) => (
                  <CommandItem key={h.key} value={h.key} onSelect={() => go(h.path)}>
                    <h.icon aria-hidden />
                    <span className="min-w-0 flex-1 truncate">{h.label}</span>
                    <span className="text-xs text-muted-foreground">{h.sublabel}</span>
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
            {q.length >= MIN_CHARS && searchRecords && pending && hits.length === 0 && <p className="px-3 py-4 text-center text-sm text-muted-foreground">Mencari...</p>}
            {q.length >= MIN_CHARS && !pending && hits.length === 0 && matchingPages.length === 0 && <CommandEmpty>Tidak ada hasil untuk &ldquo;{q}&rdquo;.</CommandEmpty>}
          </CommandList>
        </Command>
      </CommandDialog>
    </>
  );
}
