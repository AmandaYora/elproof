import { execFileSync, spawn, type ChildProcess } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import zlib from "node:zlib";
import { API_ROOT, E2E_ROOT, PIDS_FILE, STATE_FILE, TMP, WEB_ROOT, apiURL, config, webURL, type State } from "./config";
import { databaseURL, admin, sql } from "./sql";
import { seed, writeState } from "./seed";

const DEFAULT_GOOGLE_CLIENT_ID_FILE = path.join(API_ROOT, "internal/shared/config/config.go");

function log(msg: string) {
  console.log(`[e2e] ${msg}`);
}

function googleClientID(): string {
  const src = fs.readFileSync(DEFAULT_GOOGLE_CLIENT_ID_FILE, "utf8");
  const m = /DefaultGoogleClientID\s*=\s*"([^"]+)"/.exec(src);
  if (!m) throw new Error("DefaultGoogleClientID not found in config.go");
  return m[1];
}

async function waitFor(url: string, what: string, seconds = 60) {
  const deadline = Date.now() + seconds * 1000;
  for (;;) {
    try {
      const res = await fetch(url);
      if (res.ok) return;
    } catch {
      /* not up yet */
    }
    if (Date.now() > deadline) throw new Error(`${what} did not become ready at ${url}`);
    await new Promise((r) => setTimeout(r, 300));
  }
}

function spawnDetached(cmd: string, args: string[], opts: { env?: NodeJS.ProcessEnv; cwd?: string; logFile: string }): ChildProcess {
  const out = fs.openSync(opts.logFile, "a");
  const child = spawn(cmd, args, { cwd: opts.cwd, env: opts.env ?? process.env, detached: true, stdio: ["ignore", out, out] });
  child.unref();
  return child;
}

function minioAvailable(): boolean {
  try {
    execFileSync(config.minioBin, ["--version"], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

/**
 * Object storage for proof files. An explicit E2E_S3_ENDPOINT wins; otherwise a
 * throwaway MinIO (data in e2e/.tmp/minio, bucket "e2e" = a folder in it) is
 * started so payments and steps can really carry proof. Without either, files
 * cannot be stored and the seed skips them.
 */
async function ensureObjectStorage(pids: Record<string, number>): Promise<{ endpoint: string; access: string; secret: string } | null> {
  if (config.s3Endpoint) return { endpoint: config.s3Endpoint, access: config.s3Access, secret: config.s3Secret };
  if (!config.minio || !minioAvailable()) return null;
  const root = path.join(TMP, "minio");
  fs.rmSync(root, { recursive: true, force: true });
  fs.mkdirSync(path.join(root, "data", "e2e"), { recursive: true });
  const access = "e2eaccess";
  const secret = "e2esecretkey";
  const address = `127.0.0.1:${config.minioPort}`;
  log(`starting the throwaway MinIO on ${config.minioPort}`);
  const child = spawnDetached(config.minioBin, ["server", path.join(root, "data"), "--address", address, "--console-address", `127.0.0.1:${config.minioPort + 1}`], {
    env: { ...process.env, MINIO_ROOT_USER: access, MINIO_ROOT_PASSWORD: secret },
    logFile: path.join(TMP, "minio.log"),
  });
  pids.minio = child.pid!;
  await waitFor(`http://${address}/minio/health/live`, "MinIO");
  return { endpoint: address, access, secret };
}

function ensureMySQL() {
  const reachable = () => {
    try {
      sql("SELECT 1");
      return true;
    } catch {
      return false;
    }
  };
  if (reachable()) return;
  if (admin.host === "127.0.0.1" && admin.port === "3399") {
    log("starting the throwaway MySQL on 3399");
    execFileSync("sh", [path.join(E2E_ROOT, "scripts/start-mysql.sh")], { stdio: "inherit" });
    if (reachable()) return;
  }
  throw new Error(`MySQL is not reachable with E2E_MYSQL_ADMIN=${config.mysqlAdmin}`);
}

function buildBundleStats(outDir: string): State["bundle"] {
  const assets = path.join(outDir, "assets");
  const chunks: { file: string; raw: number; gzip: number }[] = [];
  for (const f of fs.readdirSync(assets).filter((n) => n.endsWith(".js"))) {
    const buf = fs.readFileSync(path.join(assets, f));
    chunks.push({ file: f, raw: buf.length, gzip: zlib.gzipSync(buf, { level: 9 }).length });
  }
  chunks.sort((a, b) => b.gzip - a.gzip);
  return { totalJsRaw: chunks.reduce((n, c) => n + c.raw, 0), totalJsGzip: chunks.reduce((n, c) => n + c.gzip, 0), chunks };
}

export function readPids(): Record<string, number> {
  try {
    return JSON.parse(fs.readFileSync(PIDS_FILE, "utf8"));
  } catch {
    return {};
  }
}

export async function stackHealthy(): Promise<boolean> {
  try {
    const [a, w] = await Promise.all([fetch(`${apiURL()}/api/v1/health`), fetch(webURL())]);
    return a.ok && w.ok && fs.existsSync(STATE_FILE);
  } catch {
    return false;
  }
}

/** Brings the whole stack up (or reuses a running one) and returns its state. */
export async function startStack(): Promise<State> {
  fs.mkdirSync(TMP, { recursive: true });
  if (config.reuse && (await stackHealthy())) {
    log("reusing the running stack");
    return JSON.parse(fs.readFileSync(STATE_FILE, "utf8"));
  }
  await stopStack({ dropDatabase: true });
  fs.mkdirSync(TMP, { recursive: true });

  ensureMySQL();
  const db = config.dbName;
  sql(`DROP DATABASE IF EXISTS \`${db}\`; CREATE DATABASE \`${db}\` CHARACTER SET utf8mb4`);

  log("building the API");
  const bin = path.join(TMP, "api-server");
  execFileSync("go", ["build", "-o", bin, "./cmd/server"], { cwd: API_ROOT, stdio: "inherit" });

  const pids: Record<string, number> = {};
  const googleURL = `http://127.0.0.1:${config.googlePort}`;
  const google = spawnDetached("node", [path.join(E2E_ROOT, "scripts/fake-google.mjs"), String(config.googlePort), googleClientID()], { logFile: path.join(TMP, "google.log") });
  pids.google = google.pid!;
  await waitFor(`${googleURL}/jwks`, "fake Google");

  const storage = await ensureObjectStorage(pids);
  const s3 = storage !== null;
  const apiEnv: NodeJS.ProcessEnv = {
    ...process.env,
    APP_ENV: "development",
    APP_PORT: String(config.apiPort),
    DATABASE_URL: databaseURL(db),
    JWT_SECRET: "e2e-jwt-secret-not-for-anything-real-0123456789",
    JWT_ACCESS_TTL: "24h",
    GOOGLE_JWKS_URL: `${googleURL}/jwks`,
    S3_ENDPOINT: storage ? storage.endpoint : "127.0.0.1:9",
    S3_BUCKET: "e2e",
    S3_USE_SSL: "false",
    S3_REGION: "us-east-1",
    S3_ACCESS_KEY: storage ? storage.access : config.s3Access,
    S3_SECRET_KEY: storage ? storage.secret : config.s3Secret,
    PAYMENT_ENCRYPTION_KEY: "e2e-payment-encryption-key-long-random-0123456789",
  };
  log("migrating and seeding the platform admin");
  execFileSync(bin, ["migrate", "up"], { cwd: TMP, env: apiEnv, stdio: "inherit" });
  execFileSync(bin, ["seed"], { cwd: TMP, env: apiEnv, stdio: "inherit" }); // one super admin + one plan, nothing else

  log("starting the API");
  const wrapper = config.apiWrapper.trim().split(/\s+/).filter(Boolean);
  const api = wrapper.length
    ? spawnDetached(wrapper[0], [...wrapper.slice(1), bin], { env: apiEnv, cwd: TMP, logFile: path.join(TMP, "api.log") })
    : spawnDetached(bin, [], { env: apiEnv, cwd: TMP, logFile: path.join(TMP, "api.log") });
  pids.api = api.pid!;
  await waitFor(`${apiURL()}/api/v1/health`, "the API");

  log("building and serving the web app (production build)");
  const outDir = path.join(TMP, "dist");
  const viteBin = [path.join(WEB_ROOT, "node_modules/.bin/vite"), path.join(WEB_ROOT, "../../node_modules/.bin/vite")].find((p) => fs.existsSync(p));
  if (!viteBin) throw new Error("vite binary not found; run npm install");
  const webEnv = { ...process.env, VITE_API_BASE_URL: apiURL() };
  execFileSync(viteBin, ["build", "--outDir", outDir, "--emptyOutDir"], { cwd: WEB_ROOT, env: webEnv, stdio: "ignore" });
  const web = spawnDetached(viteBin, ["preview", "--outDir", outDir, "--port", String(config.webPort), "--strictPort", "--host", "127.0.0.1"], {
    cwd: WEB_ROOT,
    env: webEnv,
    logFile: path.join(TMP, "web.log"),
  });
  pids.web = web.pid!;
  await waitFor(webURL(), "the web preview");
  fs.writeFileSync(PIDS_FILE, JSON.stringify(pids));

  log("seeding workspaces");
  const ids = await seed({ api: apiURL(), google: googleURL, s3, db }, webURL());
  const state: State = { apiURL: apiURL(), webURL: webURL(), storage: s3, ids, bundle: buildBundleStats(outDir) };
  writeState(STATE_FILE, state);
  log("stack ready");
  return state;
}

/** Stops every process the stack started and optionally drops its database. */
export async function stopStack(opts: { dropDatabase: boolean }) {
  const pids = readPids();
  for (const [name, pid] of Object.entries(pids)) {
    try {
      process.kill(pid, "SIGTERM");
    } catch {
      /* already gone */
    }
    void name;
  }
  // The preview server spawns its own child; free the ports if something is still listening.
  for (const port of [config.apiPort, config.webPort, config.googlePort, config.minioPort, config.minioPort + 1]) {
    try {
      const out = execFileSync("lsof", ["-tiTCP:" + port, "-sTCP:LISTEN"], { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
      for (const pid of out.split("\n").filter(Boolean)) process.kill(Number(pid), "SIGTERM");
    } catch {
      /* nothing listening */
    }
  }
  if (opts.dropDatabase) {
    try {
      sql(`DROP DATABASE IF EXISTS \`${config.dbName}\``);
    } catch {
      /* MySQL not running */
    }
  }
  fs.rmSync(PIDS_FILE, { force: true });
  fs.rmSync(STATE_FILE, { force: true });
}
