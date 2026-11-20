import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

/**
 * The SSR build of the public pages (ADR-0043) tells scripts/prerender.mjs which lazy modules a page loads while it
 * renders: every dynamic import adds its source file ("src/modules/homepage/pages/HomePage.tsx") to
 * globalThis.__prerenderImports when the prerender has set one. The prerender looks those files up in the browser build's
 * chunk map (writeChunkMap) and writes a modulepreload for their chunks, so the browser fetches them with the entry
 * script instead of after it.
 */
function recordPrerenderImports(): Plugin {
  let root = "";
  return {
    name: "elproof:record-prerender-imports",
    apply: (_, env) => Boolean(env.isSsrBuild),
    configResolved(config) {
      root = config.root;
    },
    renderDynamicImport({ targetModuleId }) {
      if (!targetModuleId) return null;
      const src = sourceOf(root, targetModuleId);
      return { left: `(globalThis.__prerenderImports?.add(${JSON.stringify(src)}), import(`, right: "))" };
    },
  };
}

/** Source path relative to the project, the way both builds name a module. */
const sourceOf = (root: string, id: string) => path.relative(root, id).split(path.sep).join("/");

/**
 * The browser build writes dist/.vite/chunks.json: which chunk holds each source module, and for each chunk the chunks it
 * imports and the style sheets it loads. Vite's own manifest names only the modules a chunk is built around, and a lazy
 * page Rollup merged into a shared chunk is missing from it. The Go server never serves a dot-folder.
 */
function writeChunkMap(): Plugin {
  let root = "";
  return {
    name: "elproof:write-chunk-map",
    apply: (_, env) => env.command === "build" && !env.isSsrBuild,
    configResolved(config) {
      root = config.root;
    },
    generateBundle(_, bundle) {
      const modules: Record<string, string> = {};
      const chunks: Record<string, { imports: string[]; css: string[] }> = {};
      let entry = "";
      for (const out of Object.values(bundle)) {
        if (out.type !== "chunk") continue;
        if (out.isEntry) entry = out.fileName;
        chunks[out.fileName] = { imports: out.imports, css: [...(out.viteMetadata?.importedCss ?? [])] };
        for (const id of out.moduleIds) if (!id.startsWith("\0")) modules[sourceOf(root, id)] = out.fileName;
      }
      this.emitFile({ type: "asset", fileName: ".vite/chunks.json", source: JSON.stringify({ entry, chunks, modules }) });
    },
  };
}

/**
 * The page's text is set in Plus Jakarta Sans (latin subset for Indonesian). Without a preload the browser finds the font
 * only after it has read the style sheet; with it, the font downloads alongside the style sheet.
 */
function preloadTextFont(): Plugin {
  return {
    name: "elproof:preload-text-font",
    apply: "build",
    transformIndexHtml: {
      order: "post",
      handler(_html, ctx) {
        const font = Object.keys(ctx.bundle ?? {}).find((f) => /plus-jakarta-sans-latin-wght-normal-[\w-]+\.woff2$/.test(f));
        if (!font) return [];
        return [
          {
            tag: "link",
            attrs: { rel: "preload", href: `/${font}`, as: "font", type: "font/woff2", crossorigin: "" },
            injectTo: "head",
          },
        ];
      },
    },
  };
}

export default defineConfig(({ isSsrBuild }) => ({
  plugins: [react(), tailwindcss(), recordPrerenderImports(), writeChunkMap(), preloadTextFont()],
  resolve: { alias: { "@": path.resolve(__dirname, "src") } },
  server: { port: 5173 },
  // The SSR build is only the build-time renderer of the public pages (npm run prerender, ADR-0043): no public/ copy.
  build: isSsrBuild ? { copyPublicDir: false } : undefined,
}));
