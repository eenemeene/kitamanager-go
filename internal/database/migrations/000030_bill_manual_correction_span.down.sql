ALTER TABLE government_funding_bill_periods
    DROP CONSTRAINT IF EXISTS correction_booking_span_ordered,
    DROP CONSTRAINT IF EXISTS correction_booking_span_complete;

ALTER TABLE government_funding_bill_periods
    DROP COLUMN IF EXISTS correction_booking_reason,
    DROP COLUMN IF EXISTS correction_booking_to_month,
    DROP COLUMN IF EXISTS correction_booking_from_month;
