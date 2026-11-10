let counter = 0;

export interface Reply {
  status: number;
  text: string;
  json: any;
}

/** One API call. Every call "comes from" a different address so per-IP limiters do not turn seeding into a rate-limit test. */
export async function call(base: string, token: string | null, method: string, path: string, body?: unknown, extraHeaders?: Record<string, string>): Promise<Reply> {
  counter += 1;
  const headers: Record<string, string> = { "X-Forwarded-For": `10.20.${Math.floor(counter / 250)}.${(counter % 250) + 1}` };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;
  Object.assign(headers, extraHeaders);
  const res = await fetch(base + path, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) });
  const text = await res.text();
  let json: any = null;
  try {
    json = JSON.parse(text);
  } catch {
    /* not JSON */
  }
  return { status: res.status, text, json };
}

export function must(what: string, r: Reply): Reply {
  if (r.status < 200 || r.status > 299) throw new Error(`seed step "${what}" failed: ${r.status} ${r.text.slice(0, 400)}`);
  return r;
}
