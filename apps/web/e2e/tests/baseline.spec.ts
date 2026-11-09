import { test } from "@playwright/test";
import { AUTH } from "../lib/config";
import { prepareContext, settle, state } from "../lib/fixtures";
import type { AuthKey } from "../lib/pages";

/**
 * Lab baseline per route on the mobile profile: 4x CPU, ~Fast 3G (1.6 Mbps down, 750 Kbps up, 150 ms RTT),
 * cold cache, production build served by `vite preview` (no compression). Counts the API requests the screen
 * makes until the network is idle and their JSON bytes, the JS it loads, LCP, CLS and a TBT-like sum of long tasks.
 * One real interaction (a navigation click) gives an INP-like figure: indicative only, not field INP.
 */
interface Route {
  id: string;
  label: string;
  auth: AuthKey;
  route: string;
  /** Click this after load to take an INP-like reading. */
  interact?: { role: "link" | "button"; name: string | RegExp };
}

const ROUTES: Route[] = [
  { id: "wo-beranda", label: "Beranda WO", auth: "owner", route: "/dashboard", interact: { role: "button", name: "Buka menu" } },
  { id: "wo-proyek", label: "Daftar Project", auth: "owner", route: "/projects" },
  { id: "wo-proyek-ringkasan", label: "Detail Proyek (Ringkasan)", auth: "owner", route: "/projects/1/ringkasan" },
  { id: "wo-proyek-vendor", label: "Detail Proyek, tab Vendor", auth: "owner", route: "/projects/1/vendor" },
  { id: "wo-proyek-keuangan", label: "Detail Proyek, tab Keuangan", auth: "owner", route: "/projects/1/pembayaran/client" },
  { id: "wo-proyek-kendala", label: "Detail Proyek, tab Kendala", auth: "owner", route: "/projects/1/kendala" },
  { id: "wo-klien", label: "Daftar Client", auth: "owner", route: "/clients" },
  { id: "wo-vendor", label: "Daftar Vendor", auth: "owner", route: "/vendors" },
  { id: "portal-beranda", label: "Beranda Portal", auth: "client", route: "/portal/beranda", interact: { role: "link", name: /Jadwal/ } },
  { id: "portal-jadwal", label: "Portal, Jadwal", auth: "client", route: "/portal/timeline" },
  { id: "portal-pembayaran", label: "Portal, Pembayaran", auth: "client", route: "/portal/pembayaran" },
  { id: "portal-vendor", label: "Portal, Vendor", auth: "client", route: "/portal/vendor" },
  { id: "portal-kendala", label: "Portal, Kendala", auth: "client", route: "/portal/kendala" },
  { id: "platform-ringkasan", label: "Ringkasan Platform", auth: "admin", route: "/platform/dashboard", interact: { role: "button", name: "Buka menu" } },
  { id: "platform-tenant", label: "Daftar Tenant", auth: "admin", route: "/platform/tenant" },
  { id: "platform-tenant-detail", label: "Detail Tenant", auth: "admin", route: "/platform/tenant/1/langganan" },
  { id: "platform-tagihan", label: "Tagihan Platform", auth: "admin", route: "/platform/transaksi" },
  { id: "marketing-beranda", label: "Beranda pemasaran", auth: "anon", route: "/" },
];

test.describe.configure({ mode: "serial" });

for (const r of ROUTES) {
  test(`baseline ${r.id}`, async ({ browser }, info) => {
    const context = await browser.newContext({ ...info.project.use, storageState: AUTH[r.auth] });
    const page = await context.newPage();
    await prepareContext(page, { freezeClock: false });
    await page.addInitScript(() => {
      const w = window as unknown as { __m: { lcp: number; cls: number; longTasks: number[]; events: number[] } };
      w.__m = { lcp: 0, cls: 0, longTasks: [], events: [] };
      const observe = (type: string, cb: (e: PerformanceEntry) => void, extra: object = {}) => {
        try {
          new PerformanceObserver((list) => list.getEntries().forEach(cb)).observe({ type, buffered: true, ...extra } as PerformanceObserverInit);
        } catch {
          /* unsupported */
        }
      };
      observe("largest-contentful-paint", (e) => (w.__m.lcp = e.startTime));
      observe("layout-shift", (e) => {
        const ls = e as unknown as { value: number; hadRecentInput: boolean };
        if (!ls.hadRecentInput) w.__m.cls += ls.value;
      });
      observe("longtask", (e) => w.__m.longTasks.push(e.duration));
      observe("event", (e) => w.__m.events.push(e.duration), { durationThreshold: 16 });
    });

    const cdp = await context.newCDPSession(page);
    await cdp.send("Network.enable");
    await cdp.send("Network.emulateNetworkConditions", { offline: false, latency: 150, downloadThroughput: (1.6 * 1024 * 1024) / 8, uploadThroughput: (750 * 1024) / 8 });
    await cdp.send("Emulation.setCPUThrottlingRate", { rate: 4 });

    const api: { method: string; path: string; bytes: number }[] = [];
    const pending: Promise<void>[] = [];
    page.on("response", (res) => {
      const url = new URL(res.url());
      if (!url.pathname.startsWith("/api/")) return;
      pending.push(
        res
          .body()
          .then((b) => void api.push({ method: res.request().method(), path: url.pathname + (url.search ? "?…" : ""), bytes: b.length }))
          .catch(() => void api.push({ method: res.request().method(), path: url.pathname, bytes: 0 }))
      );
    });

    const started = Date.now();
    await page.goto(r.route);
    await page.waitForLoadState("networkidle");
    const idleMs = Date.now() - started;
    await settle(page);
    await Promise.all(pending);

    const m = await page.evaluate(() => {
      const w = window as unknown as { __m: { lcp: number; cls: number; longTasks: number[] } };
      const js = performance
        .getEntriesByType("resource")
        .map((e) => e as PerformanceResourceTiming)
        .filter((e) => e.name.includes("/assets/") && e.name.split("?")[0].endsWith(".js"))
        .map((e) => ({ file: e.name.split("/").pop()!, transfer: e.transferSize || e.encodedBodySize }));
      return { lcp: w.__m.lcp, cls: w.__m.cls, tbt: w.__m.longTasks.reduce((n, d) => n + Math.max(0, d - 50), 0), longTasks: w.__m.longTasks.length, js };
    });

    let inpLike: number | null = null;
    if (r.interact) {
      await page.evaluate(() => ((window as unknown as { __m: { events: number[] } }).__m.events.length = 0));
      const target = page.getByRole(r.interact.role, { name: r.interact.name }).first();
      if (await target.count()) {
        // One real interaction; if it cannot be performed the reading is simply absent.
        const clicked = await target.click({ timeout: 5000 }).then(() => true, () => false);
        if (clicked) {
          await page.waitForTimeout(800);
          inpLike = await page.evaluate(() => Math.max(0, ...(window as unknown as { __m: { events: number[] } }).__m.events));
        }
      }
    }

    const bundle = state().bundle;
    const gz = new Map((bundle?.chunks ?? []).map((c) => [c.file, c.gzip]));
    const jsGzip = m.js.reduce((n, j) => n + (gz.get(j.file) ?? 0), 0);
    await info.attach("perf", {
      contentType: "application/json",
      body: JSON.stringify({
        id: r.id,
        label: r.label,
        route: r.route,
        apiCalls: api.length,
        apiBytes: api.reduce((n, a) => n + a.bytes, 0),
        apiRequests: api.map((a) => `${a.method} ${a.path} (${a.bytes} B)`),
        jsChunks: m.js.length,
        jsTransfer: m.js.reduce((n, j) => n + j.transfer, 0),
        jsGzip,
        lcpMs: Math.round(m.lcp),
        cls: Number(m.cls.toFixed(3)),
        tbtMs: Math.round(m.tbt),
        longTasks: m.longTasks,
        networkIdleMs: idleMs,
        inpLikeMs: inpLike === null ? null : Math.round(inpLike),
        bundle: bundle ? { totalJsRaw: bundle.totalJsRaw, totalJsGzip: bundle.totalJsGzip, chunks: bundle.chunks } : null,
      }),
    });
    await context.close();
  });
}
