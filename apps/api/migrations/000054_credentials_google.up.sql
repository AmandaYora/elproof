-- Sign in with Google (PLAN.md Fase 2, decisions K2/K5). A WO's Owner and
-- staff sign in with their Google account: google_sub is Google's stable,
-- never-reassigned account id ("sub" claim), unique across ElProof. Such an
-- account has no ElProof password, so password_hash becomes nullable.
ALTER TABLE credentials
  MODIFY password_hash VARCHAR(255) NULL,
  ADD COLUMN google_sub VARCHAR(255) NULL AFTER email,
  ADD UNIQUE KEY uq_credentials_google_sub (google_sub);
