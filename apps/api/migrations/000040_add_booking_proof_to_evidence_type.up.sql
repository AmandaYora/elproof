-- 'Booking Proof' -- bukti booked on a project_vendors engagement, added to
-- "Ubah Vendor Project" (PLAN.md revisi-timeline-vendor-role-sales).
ALTER TABLE evidence MODIFY COLUMN type
  ENUM('Quotation', 'Invoice', 'Contract', 'Transfer Proof', 'Receipt', 'Purchase Order', 'Photo', 'Document', 'Screenshot', 'Minutes of Meeting', 'Other', 'Booking Proof') NOT NULL;
