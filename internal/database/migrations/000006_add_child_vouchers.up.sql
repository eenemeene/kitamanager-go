CREATE TABLE child_vouchers (
    id BIGSERIAL PRIMARY KEY,
    child_id BIGINT NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    voucher_number VARCHAR(17) NOT NULL,
    first_seen DATE NOT NULL,
    created_at TIMESTAMPTZ,
    CONSTRAINT uni_child_vouchers_voucher_number UNIQUE (voucher_number)
);

CREATE INDEX idx_child_vouchers_child_id ON child_vouchers(child_id);

-- Populate from existing contract data
INSERT INTO child_vouchers (child_id, voucher_number, first_seen)
SELECT DISTINCT ON (cc.child_id, cc.voucher_number)
    cc.child_id,
    cc.voucher_number,
    cc.from_date
FROM child_contracts cc
WHERE cc.voucher_number IS NOT NULL
    AND cc.voucher_number != ''
    AND cc.voucher_number != '000'
    AND cc.voucher_number != 'abc'
    AND cc.voucher_number != 'def'
ON CONFLICT (voucher_number) DO NOTHING;

-- Drop voucher from contracts
ALTER TABLE child_contracts DROP COLUMN IF EXISTS voucher_number;
