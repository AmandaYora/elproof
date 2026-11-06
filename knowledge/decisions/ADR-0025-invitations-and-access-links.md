# ADR-0025: Invitations and one-time access links (no one sets another person's password)

## Status
Accepted — implemented 2026-11-06 (PLAN.md Fase 3, decision K5).

## Context
Owners typed passwords for their staff and clients, ElProof's team typed the password of every
sales-assisted Owner, and those passwords were shown on screen to be passed on. OWASP ASVS 5.0
(§6.4.1, §6.4.6) says nobody should set another person's password. Decision K5: WO staff sign in
with Google only; clients keep username + password but choose the password themselves.

## Decision
- **One-time access links** (`identity.auth_tokens`): 256-bit random token, only its SHA-256 hash
  stored; single use (`used_at`, redeemed in one transaction with the account change); expiring;
  issuing a new link revokes every open link of the same login; deactivating a login revokes them
  too. Purposes: `staff_invite` (7 days), `client_invite` (7 days), `client_reset` (24 hours).
- **Pending logins**: an invited person's login exists from the start (reserving username and
  email, carrying the role) but has no password and no Google account, so nothing can sign in
  with it until the link is redeemed.
- **Staff / Owner invitation**: accepted at `/undangan/{token}` by signing in with the Google
  account whose verified email equals the invited email — a forwarded link is useless to anyone
  else. Accepting re-links the login to that Google account (also the recovery path for a lost
  Google account) and ends older sessions. Self-service Owners (ADR-0024) and Owners registered by
  ElProof's team (`POST /tenants` now returns an invitation; `POST /tenants/{id}/owner-invite`
  replaces the password reset) use the same mechanism. **Staff password login is closed**
  (`use_google`); legacy staff with a password link Google once via ADR-0024's confirmation.
- **Client invitation / reset**: the client sets their own password (≥ 8 characters, not equal to
  the username); a reset ends every other session. `POST /clients/{id}/access-link` issues an
  invitation while no password was ever set, otherwise a reset. Replacing a family representative
  clears the previous person's password, sessions and links, renames the login after the new
  person, and returns their invitation. The old "WO types a new password" endpoint is gone.
- **Delivery**: no e-mail or WhatsApp API — the WO console shows the link with "Salin link" and a
  ready-made `wa.me` message sent from the WO's own WhatsApp (PLAN.md §2.7).
- Staff and client lists show `loginStatus` (`invited` until the link is redeemed).

## Consequences
- An invited person who never opens the link occupies a trial seat until removed or deactivated.
- Links are bearer secrets for their lifetime: they travel through WhatsApp, which the WO
  controls; the short reset lifetime and single use limit exposure.
- Email changes on a staff or client row are synced to the login (an invitation is accepted with
  the current email). An Owner re-invitation goes to the email on the Owner's login, never to the
  Platform Console's editable tenant contact email (otherwise a Support account could redirect it).
- Deleting a staff member or client (or rolling back a failed invitation) deletes the login with
  its sessions and links, freeing the email and username.
- A re-invitation cuts off the old Google account only when accepted; to cut access at once,
  deactivate the member first.
- Link expiry is judged with the app's clock; a frozen workspace refuses before a link is spent.
- Tokens appear in URL paths (`/undangan/…`, `/api/v1/auth/links/…`): keep them out of shared
  logs (see DEPLOYMENT.md). No `Referrer-Policy: no-referrer` — Google Identity Services needs
  the page origin; the browser default sends only the origin.
