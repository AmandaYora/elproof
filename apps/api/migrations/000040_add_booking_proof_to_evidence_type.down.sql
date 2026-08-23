ALTER TABLE evidence MODIFY COLUMN type
  ENUM('Quotation', 'Invoice', 'Contract', 'Transfer Proof', 'Receipt', 'Purchase Order', 'Photo', 'Document', 'Screenshot', 'Minutes of Meeting', 'Other') NOT NULL;
