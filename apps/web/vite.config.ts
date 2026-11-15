import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

export default defineConfig(({ isSsrBuild }) => ({
  plugins: [react(), tailwindcss()],
  resolve: { alias: { "@": path.resolve(__dirname, "src") } },
  server: { port: 5173 },
  // The SSR build is only the build-time renderer of the public pages (npm run prerender, ADR-0042): no public/ copy.
  build: isSsrBuild ? { copyPublicDir: false } : undefined,
}));
