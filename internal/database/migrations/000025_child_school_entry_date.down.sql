-- Reverses 000025. Any recorded Zurückstellung is lost, and those children go
-- back to being told they leave on the computed date.

ALTER TABLE children DROP COLUMN IF EXISTS school_entry_date;
