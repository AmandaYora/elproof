import { execFileSync } from "node:child_process";
import { config } from "./config";

function parseAdmin() {
  // user[:password]@host:port
  const m = /^([^:@]+)(?::([^@]*))?@([^:]+):(\d+)$/.exec(config.mysqlAdmin);
  if (!m) throw new Error(`E2E_MYSQL_ADMIN must look like user[:pass]@host:port, got "${config.mysqlAdmin}"`);
  return { user: m[1], pass: m[2] ?? "", host: m[3], port: m[4] };
}

export const admin = parseAdmin();

/** Runs SQL through the mysql CLI (no driver needed). Returns tab-separated rows without headers. */
export function sql(statement: string, database?: string): string {
  const args = ["-h", admin.host, "-P", admin.port, "-u", admin.user, "-N", "-B", "--default-character-set=utf8mb4"];
  if (admin.pass) args.push(`-p${admin.pass}`);
  if (database) args.push(database);
  args.push("-e", statement);
  return execFileSync(config.mysqlBin, args, { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trim();
}

/** The DATABASE_URL form the API reads. */
export function databaseURL(db: string): string {
  return `mysql://${admin.user}:${admin.pass}@tcp(${admin.host}:${admin.port})/${db}`;
}
