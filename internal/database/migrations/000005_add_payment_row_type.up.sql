ALTER TABLE government_funding_bill_payments
    ADD COLUMN row_type VARCHAR(20) NOT NULL DEFAULT 'regular';
