# ADR-0044: Refresh token in an HttpOnly cookie, rotation families, and browser security headers

## Status
Accepted (2026-11-15). Fixes security QA findings **EP-07** (tokens in `localStorage`) and **EP-08** (no HTTP
security headers, server version visible). Amends ADR-0005 (session model) and the support-session flow of ADR-0030.

## Context
- **EP-07.** After sign-in the web app stored the whole session, access token *and* refresh token, in
  `localStorage["elproof.session"]`. Any script running in the page (an XSS, a compromised dependency, a browser
  extension) could read the refresh token and keep a session alive elsewhere for its whole lifetime (168 h).
  `POST /auth/refresh` and `/auth/logout` took the refresh token from the JSON body.
- Rotation existed (each refresh revoked the presented token and issued a new one), but a revoked token presented
  again was simply refused: nothing told the server that two parties held tokens of one sign-in.
- **EP-08.** Responses carried no Content-Security-Policy, HSTS, `X-Frame-Options`/`frame-ancestors`,
  `X-Content-Type-Options` or `Referrer-Policy`, and the host nginx announced `Server: nginx/1.24.0 (Ubuntu)`.

## Decision

### Refresh token: cookie only
- Every endpoint that starts a session (`/auth/login`, `/auth/mfa/verify`, `/auth/mfa/enable`, `/auth/google`,
  `/auth/google/link`, `/auth/links/{token}/google|password`) and `/auth/refresh` set the refresh token as a cookie and
  **no longer return it in the body**:
  `__Secure-elproof_refresh=<token>; Path=/api/v1/auth; Max-Age=<JWT_REFRESH_TTL>; HttpOnly; Secure; SameSite=Strict`.
  - `HttpOnly`: no page script can read it — the point of EP-07.
  - `Secure` + the `__Secure-` prefix: https only, and a browser refuses the name from an insecure page. With
    `APP_ENV=development` (plain http on localhost) the cookie is named `elproof_refresh` and not Secure.
  - `SameSite=Strict`: the cookie only ever needs to go with the SPA's own XHR to the same site. Google sign-in uses
    the GIS JavaScript callback (no redirect back to us), the payment gateway's return is a top-level navigation that
    does not need the cookie (the page then refreshes with its own same-site XHR), so no flow needs `Lax`.
  - `Path=/api/v1/auth`: the cookie is not sent with ordinary API calls, file requests or page loads.
  - `Max-Age` = the refresh token's TTL; every rotation re-issues it with a new full lifetime (sliding session, as
    before).
- `/auth/refresh` and `/auth/logout` read **only** the cookie. No transition period for a body token: the only
  client is our SPA, which changes in the same release; Postman keeps cookies in its jar. Sessions stored by the old
  web app cannot be migrated (the browser has no cookie yet), so after this release everyone signs in once more; the
  web app deletes the old `elproof.session` key on first load.
- A refused refresh (any 4xx) clears the cookie; a 5xx keeps it for the next try. Logout clears it.

### Rotation families and reuse detection
- `refresh_tokens.family_id` (migration `000076`) links every rotated token to the first token of its sign-in
  (`NULL` = the root itself). Refresh stores the successor **before** revoking the presented token, so a concurrent
  request never finds a family without a live token.
- A token that was already rotated is accepted again only within `REFRESH_REUSE_GRACE` (default 30 s) and only while
  its family still has a live token (not signed out, not revoked). This covers two tabs refreshing with the same
  cookie and a response lost on its way back, without a cross-tab lock in the browser.
- Presented later than the grace, the rotated token is treated as stolen: the **whole family** is revoked (the
  attacker's and the user's tokens alike; other devices of the same account — other families — are unaffected) and
  logged (`identity: rotated refresh token reused`).
- Logout revokes the whole family of the cookie's token, so a token rotated a moment ago in another tab cannot revive
  it. `RevokeAllForCredential` (deactivation, two-factor enabled, password set) is unchanged.
- `REFRESH_REUSE_GRACE` outside development is capped at 2 minutes (`cmd/server/security.go`). The end-to-end stack
  sets 12 h because many browser contexts start from one stored session.

### CSRF
The refresh cookie makes `/auth/refresh` and `/auth/logout` cookie-authenticated, so they are protected twice:
`SameSite=Strict`, and `middleware.SameOrigin` — they **require `X-Requested-With`** (a cross-site page can only add a
custom header through CORS, which production never grants) and refuse requests a browser marks as cross-site
(`Sec-Fetch-Site: cross-site`, or an `Origin` other than the request's `Host`/`X-Forwarded-Host`). The sign-in
endpoints get the same cross-site refusal without the header requirement, so another site cannot sign a visitor into
an account of its choosing (login CSRF) while server-to-server callers (no `Origin`) keep working. Refusal: 403
`csrf_refused`. In development any loopback origin counts as ours (Vite on another port). The dev CORS middleware now
allows `X-Requested-With` and sends `Access-Control-Allow-Credentials: true` for loopback origins only.

### Web app
- The access token lives only in memory (`useAuthStore`). `localStorage["elproof.profile"]` holds a token-free
  profile for the first paint. After a load the HTTP client refreshes (cookie) before the first call.
- Support sessions (read-only Owner token, no refresh token): the token crosses the full page load that starts the
  session in `sessionStorage` and is deleted when the new page reads it. A reload during a support session ends it
  (the token is gone): the admin is returned to the workspace's page and the API is told. The admin's own refresh
  token stays in its HttpOnly cookie, so the tenant-written WO Console never sees a long-lived admin credential (the
  reason the old code stripped it from `returnTo`).
- Accepted limitation: a script running in our origin can still *use* the session while the page is open (call
  `/auth/refresh` itself). HttpOnly removes theft of the long-lived token; the Content-Security-Policy below is the
  defence against the script running at all.

### Security headers (EP-08)
`middleware.SecurityHeaders` wraps the whole server: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`,
`Referrer-Policy: strict-origin-when-cross-origin`, `Permissions-Policy` (camera, microphone, geolocation, payment,
usb, serial, bluetooth, browsing-topics off — photos are picked with a file input, which needs none),
`Cross-Origin-Opener-Policy: same-origin-allow-popups` (keeps Google's sign-in popup working), HSTS
`max-age=31536000; includeSubDomains` outside development, and a CSP:

- `/api/*`: `default-src 'none'; frame-ancestors 'none'`.
- the app (built from config so the real storage host is named):
  `default-src 'self'; script-src 'self' https://accounts.google.com/gsi/client;
  style-src 'self' 'unsafe-inline' https://accounts.google.com/gsi/style;
  img-src 'self' data: blob: https://tripay.co.id <storage>; font-src 'self' data:;
  connect-src 'self' https://accounts.google.com/gsi/ <storage>; frame-src 'self' blob: https://accounts.google.com/gsi/ <storage>;
  media-src 'self' blob:; worker-src 'self'; manifest-src 'self'; object-src 'none'; base-uri 'self';
  form-action 'self'; frame-ancestors 'none'`
  where `<storage>` is `https://<S3_ENDPOINT>` and `https://*.<S3_ENDPOINT>` (path- and virtual-host-style presigned
  links) when `S3_PRESIGN=true`. No `'unsafe-inline'`/`'unsafe-eval'` for scripts (the built `index.html` has no inline
  script). `'unsafe-inline'` for styles because the UI kit and the toaster inject `<style>` at runtime; CSS cannot run
  script.

nginx: the repo's `infra/nginx/nginx.conf` gets `server_tokens off` and passes the app's headers through. The host
nginx on the VPS is outside this repo: `docs/DEPLOYMENT.md` §6 lists what the operator changes there.

## Consequences
- Everyone signs in again once after the release (old `localStorage` sessions carry no cookie).
- A new third-party script, image host, iframe or upload target needs a CSP change in `AppContentSecurityPolicy` and
  its test, or the browser blocks it. A pre-rendered page that adds an inline `<script>` needs a hash or nonce.
- Local development: the page and the API must use the same host name (`localhost` and `localhost`, not `localhost`
  and `127.0.0.1`) for the `SameSite=Strict` cookie to be sent.
- e2e: stored browser states carry the refresh cookie plus the profile; specs that need an access token call
  `browserSession(page)` (a refresh from inside the page).
