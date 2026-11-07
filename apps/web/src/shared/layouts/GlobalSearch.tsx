import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { FolderKanban, Store, Users } from "lucide-react";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

interface ResultItem {
  key: string;
  label: string;
  sublabel: string;
  icon: typeof FolderKanban;
  path: string;
}

const PER_KIND = 4;
const MIN_CHARS = 2;

interface Named {
  id: number | string;
  name: string;
  status?: string;
}

/** One kind of result: a small, server-searched page, never the whole list. */
async function searchKind(url: string, q: string, signal: AbortSignal): Promise<Named[]> {
  const res = await httpClient.get(url, { params: { search: q, limit: PER_KIND }, signal });
  return res.data.data as Named[];
}

export function GlobalSearch() {
  const [query, setQuery] = useState("");
  const [focused, setFocused] = useState(false);
  const [results, setResults] = useState<ResultItem[]>([]);
  const [searching, setSearching] = useState(false);
  const navigate = useNavigate();
  const containerRef = useRef<HTMLDivElement>(null);
  const debounced = useDebouncedValue(query.trim(), 250);

  useEffect(() => {
    if (debounced.length < MIN_CHARS) {
      setResults([]);
      setSearching(false);
      return;
    }
    const controller = new AbortController();
    setSearching(true);
    // Each kind fails on its own: a Wedding Planner gets 403 for tenant-wide
    // clients (Owner/Admin only, PLAN.md RBAC), and search must still show
    // projects and vendors rather than nothing.
    const settle = <T,>(p: Promise<T>) => p.then((value) => ({ ok: true as const, value }), () => ({ ok: false as const }));
    Promise.all([
      settle(searchKind(API.projects.base, debounced, controller.signal)),
      settle(searchKind(API.vendors.base, debounced, controller.signal)),
      settle(searchKind(API.clients.base, debounced, controller.signal)),
    ]).then(([projects, vendors, clients]) => {
      if (controller.signal.aborted) return;
      const items: ResultItem[] = [
        ...(projects.ok ? projects.value : []).map((p) => ({ key: `p-${p.id}`, label: p.name, sublabel: `Project${p.status ? ` · ${p.status}` : ""}`, icon: FolderKanban, path: ROUTE_PATHS.projectDetail(String(p.id)) })),
        ...(vendors.ok ? vendors.value : []).map((v) => ({ key: `v-${v.id}`, label: v.name, sublabel: "Vendor", icon: Store, path: ROUTE_PATHS.vendors })),
        ...(clients.ok ? clients.value : []).map((c) => ({ key: `c-${c.id}`, label: c.name, sublabel: "Client", icon: Users, path: ROUTE_PATHS.clients })),
      ];
      setResults(items.slice(0, 8));
      setSearching(false);
    });
    return () => controller.abort();
  }, [debounced]);

  return (
    <div className="relative w-full max-w-sm" ref={containerRef}>
      <SearchInput
        placeholder="Cari project, vendor, atau client..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onFocus={() => setFocused(true)}
        onBlur={() => setTimeout(() => setFocused(false), 120)}
      />
      {focused && query.trim().length >= 2 && (
        <div className="absolute left-0 right-0 top-11 z-30 overflow-hidden rounded-md border border-border bg-white shadow-lg">
          {searching || query.trim() !== debounced ? (
            <p className="px-4 py-3 text-[13px] text-text-secondary">Mencari...</p>
          ) : results.length === 0 ? (
            <p className="px-4 py-3 text-[13px] text-text-secondary">Tidak ada hasil untuk &ldquo;{query}&rdquo;.</p>
          ) : (
            <ul>
              {results.map((r) => (
                <li key={r.key}>
                  <button
                    onClick={() => {
                      navigate(r.path);
                      setQuery("");
                    }}
                    className="flex w-full items-center gap-3 px-4 py-2.5 text-left text-sm hover:bg-surface-muted"
                  >
                    <r.icon className="h-4 w-4 shrink-0 text-text-secondary" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium text-text-primary">{r.label}</span>
                      <span className="block truncate text-[12px] text-text-secondary">{r.sublabel}</span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
