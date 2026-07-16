import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { FolderKanban, Store, Users } from "lucide-react";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useClientStore } from "@/modules/clients/stores/useClientStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

interface ResultItem {
  key: string;
  label: string;
  sublabel: string;
  icon: typeof FolderKanban;
  path: string;
}

export function GlobalSearch() {
  const [query, setQuery] = useState("");
  const [focused, setFocused] = useState(false);
  const navigate = useNavigate();
  const containerRef = useRef<HTMLDivElement>(null);

  const projects = useProjectStore((s) => s.projects);
  const fetchProjects = useProjectStore((s) => s.fetchProjects);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const allClients = useClientStore((s) => s.allClients);
  const fetchAllClients = useClientStore((s) => s.fetchAllClients);

  useEffect(() => {
    void fetchProjects();
    void fetchVendors();
    void fetchAllClients();
  }, [fetchProjects, fetchVendors, fetchAllClients]);

  const results = useMemo<ResultItem[]>(() => {
    const q = query.trim().toLowerCase();
    if (q.length < 2) return [];

    const projectResults: ResultItem[] = projects
      .filter((p) => p.name.toLowerCase().includes(q) || p.brideName.toLowerCase().includes(q) || p.groomName.toLowerCase().includes(q))
      .slice(0, 4)
      .map((p) => ({ key: `p-${p.id}`, label: p.name, sublabel: `Project · ${p.status}`, icon: FolderKanban, path: ROUTE_PATHS.projectDetail(p.id) }));

    const vendorResults: ResultItem[] = vendors
      .filter((v) => v.name.toLowerCase().includes(q))
      .slice(0, 4)
      .map((v) => ({ key: `v-${v.id}`, label: v.name, sublabel: "Vendor", icon: Store, path: ROUTE_PATHS.vendors }));

    const clientResults: ResultItem[] = allClients
      .filter((c) => c.name.toLowerCase().includes(q))
      .slice(0, 4)
      .map((c) => ({ key: `c-${c.id}`, label: c.name, sublabel: "Client", icon: Users, path: ROUTE_PATHS.clients }));

    return [...projectResults, ...vendorResults, ...clientResults].slice(0, 8);
  }, [query, projects, vendors, allClients]);

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
          {results.length === 0 ? (
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
