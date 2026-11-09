import { useCallback, useEffect, useState } from "react";
import { Info, TriangleAlert, Wrench, X } from "lucide-react";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { cn } from "@/shared/lib/cn";

interface ActiveAnnouncement {
  id: number;
  title: string;
  body: string;
  kind: "info" | "warning" | "maintenance";
}

const STYLE = {
  info: { icon: Info, box: "border-info/30 bg-info-soft text-info-text" },
  warning: { icon: TriangleAlert, box: "border-warning/30 bg-warning-soft text-warning-text" },
  maintenance: { icon: Wrench, box: "border-navy-900/20 bg-navy-900/10 text-navy-900" },
} as const;

const DISMISSED_KEY = "elproof.dismissedAnnouncements";

function readDismissed(): number[] {
  try {
    return JSON.parse(localStorage.getItem(DISMISSED_KEY) ?? "[]") as number[];
  } catch {
    return [];
  }
}

/** Platform announcements for this workspace; each can be dismissed on this browser. */
export function AnnouncementBanner() {
  const [items, setItems] = useState<ActiveAnnouncement[]>([]);
  const [dismissed, setDismissed] = useState<number[]>(readDismissed);

  useEffect(() => {
    let cancelled = false;
    httpClient
      .get(API.workspace.activeAnnouncements)
      .then((res) => {
        if (!cancelled) setItems(res.data.data as ActiveAnnouncement[]);
      })
      .catch(() => undefined); // an announcement is never worth an error screen
    return () => {
      cancelled = true;
    };
  }, []);

  const dismiss = useCallback((id: number) => {
    setDismissed((prev) => {
      const next = [...prev, id];
      try {
        localStorage.setItem(DISMISSED_KEY, JSON.stringify(next));
      } catch {
        // storage unavailable: it just comes back next visit
      }
      return next;
    });
  }, []);

  const visible = items.filter((a) => !dismissed.includes(a.id));
  if (visible.length === 0) return null;
  return (
    <div className="mb-4 flex flex-col gap-2 print:hidden">
      {visible.map((a) => {
        const { icon: Icon, box } = STYLE[a.kind] ?? STYLE.info;
        return (
          <div key={a.id} role="status" className={cn("flex items-start gap-3 rounded-lg border px-4 py-3", box)}>
            <Icon className="mt-0.5 h-4 w-4 shrink-0" />
            <div className="min-w-0 flex-1 text-[13px]">
              <p className="font-semibold">{a.title}</p>
              <p className="mt-0.5 whitespace-pre-line text-text-primary/80">{a.body}</p>
            </div>
            <button
              type="button"
              onClick={() => dismiss(a.id)}
              aria-label="Tutup pengumuman"
              className="shrink-0 rounded p-1 hover:bg-black/5"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        );
      })}
    </div>
  );
}
