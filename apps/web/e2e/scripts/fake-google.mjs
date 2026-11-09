// A fake Google for the e2e stack: serves a JWKS and mints ID tokens the API accepts
// (it is started with GOOGLE_JWKS_URL pointing here, only allowed outside production).
//   GET /jwks
//   GET /token?email=..&name=..&sub=..   -> raw token text
import crypto from "node:crypto";
import http from "node:http";

const port = Number(process.argv[2] ?? 18099);
const AUDIENCE = process.argv[3];
const kid = "e2e-1";
const { publicKey, privateKey } = crypto.generateKeyPairSync("rsa", { modulusLength: 2048 });
const jwk = { ...publicKey.export({ format: "jwk" }), kid, alg: "RS256", use: "sig" };
const b64 = (b) => Buffer.from(b).toString("base64url");

function mint(email, name, sub) {
  const now = Math.floor(Date.now() / 1000);
  const head = b64(JSON.stringify({ alg: "RS256", typ: "JWT", kid }));
  const body = b64(
    JSON.stringify({ iss: "https://accounts.google.com", aud: AUDIENCE, sub, email, email_verified: true, name, iat: now, exp: now + 7200 })
  );
  const sig = crypto.sign("RSA-SHA256", Buffer.from(`${head}.${body}`), privateKey);
  return `${head}.${body}.${b64(sig)}`;
}

http
  .createServer((req, res) => {
    const url = new URL(req.url, "http://x");
    if (url.pathname === "/jwks") {
      res.setHeader("content-type", "application/json");
      res.end(JSON.stringify({ keys: [jwk] }));
    } else if (url.pathname === "/token") {
      const q = url.searchParams;
      res.end(mint(q.get("email") ?? "owner@example.com", q.get("name") ?? "Owner", q.get("sub") ?? "g-sub-1"));
    } else {
      res.statusCode = 404;
      res.end();
    }
  })
  .listen(port, "127.0.0.1", () => console.log(`fake google on ${port}`));
