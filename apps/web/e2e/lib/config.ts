import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));

export const E2E_ROOT = path.resolve(here, "..");
export const WEB_ROOT = path.resolve(E2E_ROOT, "..");
export const API_ROOT = path.resolve(WEB_ROOT, "../api");
export const TMP = path.join(E2E_ROOT, ".tmp");
export const AUTH_DIR = path.join(TMP, "auth");
export const STATE_FILE = path.join(TMP, "state.json");
export const PIDS_FILE = path.join(TMP, "pids.json");

const env = process.env;

export const config = {
  apiPort: Number(env.E2E_API_PORT ?? 18091),
  webPort: Number(env.E2E_WEB_PORT ?? 14173),
  googlePort: Number(env.E2E_GOOGLE_PORT ?? 18099),
  mysqlAdmin: env.E2E_MYSQL_ADMIN ?? "root@127.0.0.1:3399",
  mysqlBin: env.E2E_MYSQL_BIN ?? "mysql",
  dbName: env.E2E_DB_NAME ?? "elproof_e2e_pw",
  reuse: env.E2E_REUSE === "1",
  s3Endpoint: env.E2E_S3_ENDPOINT ?? "",
  s3Access: env.E2E_S3_ACCESS_KEY ?? "x",
  s3Secret: env.E2E_S3_SECRET_KEY ?? "x",
  /** Optional command prefix for the API, e.g. `faketime -f "@2026-11-09 03:00:00"`, to freeze the server clock. */
  apiWrapper: env.E2E_API_WRAPPER ?? "",
};

export const apiURL = () => `http://127.0.0.1:${config.apiPort}`;
export const webURL = () => `http://127.0.0.1:${config.webPort}`;

/** The day every seed date is anchored to, and the instant the browser clock is fixed at (10:00 WIB). */
export const BASE_DAY = "2026-11-09";
export const FIXED_NOW = "2026-11-09T10:00:00+07:00";

export interface Session {
  accessToken: string;
  refreshToken: string;
  principalType: string;
  principalId: string;
  tenantId: number | null;
  role: string;
  displayName: string;
  mfa?: boolean;
  mfaSetupRequired?: boolean;
}

export interface State {
  apiURL: string;
  webURL: string;
  ids: {
    project: number; // "Citra & Bima": overdue installment, the richest one
    projectDone: number;
    projectDraft: number;
    tenantContoh: number;
    tenantBerbayar: number;
    tenantKosong: number;
    vendor: number;
    engagement: number;
    transaction: number;
  };
  bundle?: { totalJsRaw: number; totalJsGzip: number; chunks: { file: string; raw: number; gzip: number }[] };
}

export const AUTH = {
  owner: path.join(AUTH_DIR, "owner.json"),
  client: path.join(AUTH_DIR, "client.json"),
  admin: path.join(AUTH_DIR, "admin.json"),
  empty: path.join(AUTH_DIR, "empty.json"),
  anon: path.join(AUTH_DIR, "anon.json"),
};
