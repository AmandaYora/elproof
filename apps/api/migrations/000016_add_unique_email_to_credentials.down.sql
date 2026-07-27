ALTER TABLE credentials DROP KEY uq_credentials_email;
ALTER TABLE credentials ADD KEY idx_credentials_email (email);
