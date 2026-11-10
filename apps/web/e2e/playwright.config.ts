import { defineConfig } from "@playwright/test";
import { webURL } from "./lib/config";

const ANDROID_UA =
  "Mozilla/5.0 (Linux; Android 13; Redmi Note 12) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Mobile Safari/537.36";

const GOLDEN = /(wo|portal|platform|marketing|axe)\.spec\.ts$/;

export default defineConfig({
  testDir: "./tests",
  // One image per page, no OS suffix: CI uses a single environment (the Playwright Docker image).
  // Screenshots taken on macOS differ slightly from Linux ones, see README.
  snapshotPathTemplate: "{testDir}/../snapshots/{arg}{ext}",
  outputDir: "./.tmp/test-results",
  timeout: 90_000,
  expect: { toHaveScreenshot: { maxDiffPixelRatio: 0.01, animations: "disabled", caret: "hide", scale: "css" } },
  fullyParallel: true,
  workers: Number(process.env.E2E_WORKERS ?? 3),
  retries: 0,
  reporter: [["list"], ["./reporters/summary-reporter.ts"]],
  globalSetup: "./global-setup.ts",
  globalTeardown: "./global-teardown.ts",
  use: {
    baseURL: webURL(),
    locale: "id-ID",
    timezoneId: "Asia/Jakarta",
    reducedMotion: "reduce",
    colorScheme: "light",
    serviceWorkers: "block",
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
  },
  projects: [
    { name: "desktop", testMatch: GOLDEN, use: { viewport: { width: 1280, height: 800 } } },
    {
      name: "mobile",
      testMatch: GOLDEN,
      use: { viewport: { width: 375, height: 812 }, isMobile: true, hasTouch: true, deviceScaleFactor: 1, userAgent: ANDROID_UA },
    },
    {
      // Behavior (overlays, proof stamps, call budgets): functional, no screenshots.
      name: "behavior",
      testMatch: /(overlays|proof|forms|overview|lists)\.spec\.ts$/,
      use: { viewport: { width: 1280, height: 800 } },
    },
    {
      // Tests that WRITE data (new project, new payment). They run after every screenshot project has finished, so the
      // data they add can never leak into a golden screenshot taken at the same time.
      name: "mutating",
      testMatch: /(payment-flow|tahap5|tahap6)\.spec\.ts$/,
      dependencies: ["desktop", "mobile"],
      use: { viewport: { width: 1280, height: 800 } },
    },
    {
      // Lab measurements on the mobile profile (4x CPU, ~Fast 3G). Run alone: `npm run e2e -- --project=baseline`.
      name: "baseline",
      testMatch: /baseline\.spec\.ts$/,
      workers: 1,
      fullyParallel: false,
      use: { viewport: { width: 375, height: 812 }, isMobile: true, hasTouch: true, deviceScaleFactor: 1, userAgent: ANDROID_UA },
    },
  ],
});
