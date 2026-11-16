-- RT-08: an audited request may carry the sentence its handler stated (a toggle
-- names its resulting state and the object: "Menonaktifkan paket Pro"). Empty for
-- most routes and for every row recorded before this column; the Audit Log then
-- builds the sentence from the route as before.
ALTER TABLE audit_log ADD COLUMN description VARCHAR(255) NOT NULL DEFAULT '' AFTER action;
