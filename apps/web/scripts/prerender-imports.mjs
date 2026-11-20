// Worker of scripts/prerender.mjs: renders one public page in a module graph of its own and answers with the source files
// of the lazy modules that page loaded (recorded by the SSR build, see recordPrerenderImports in vite.config.ts). A worker
// per page, because a lazy module loads once per graph: in a shared one, the second page would never report the layout
// the first page already loaded.
import { parentPort, workerData } from "node:worker_threads";

globalThis.__prerenderImports = new Set();
const { render } = await import(workerData.entry);
await render(workerData.path);
parentPort.postMessage([...globalThis.__prerenderImports]);
