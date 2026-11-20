-- Profil usaha › Penawaran (paket-wo PLAN §3.3): the WO's own document code for
-- quotation numbers (KODE-PNW-YYMMDD-NNN), whether it charges PPN (PKP), how long
-- a new quotation stays valid, and its default terms. One row per tenant in the
-- existing table. An empty document_code means "not set yet": the platform
-- service fills it from the business name the first time it is read.
ALTER TABLE tenant_business_details
  ADD COLUMN document_code VARCHAR(5) NOT NULL DEFAULT '' AFTER bank_account_holder,
  ADD COLUMN is_pkp TINYINT(1) NOT NULL DEFAULT 0 AFTER document_code,
  ADD COLUMN default_quote_validity_days SMALLINT UNSIGNED NOT NULL DEFAULT 14 AFTER is_pkp,
  ADD COLUMN default_terms_text TEXT NULL AFTER default_quote_validity_days;
