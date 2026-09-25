-- The month a bill row is ABOUT, as opposed to the month the bill arrived in.
--
-- The ISBJ Senatsabrechnung's Vertrag sheet carries a merged "Monat/ Typ"
-- header spanning two columns: Typ is "A" (Abrechnung, regular) or "K"
-- (Korrektur), and Monat is MM.YY. We have always kept the first and thrown
-- away the second, even though they come out of the same header cell and are
-- therefore available exactly as often as each other.
--
-- They are not the same month. One bill routinely looks like this:
--
--   K  01.25     1.02 EUR
--   K  02.25     1.02 EUR
--   K  03.25     1.02 EUR
--   A  04.25   946.50 EUR
--
-- The Senate is saying: here is April, and the rate we paid you in January,
-- February and March was 1.02 EUR/month short each. Corrections arrive this
-- way whenever the Kostenblatt is republished retroactively, an
-- Integrationsstatus is approved with an effective date in the past, or an NdH
-- flag is fixed after the fact.
--
-- Without this column every aggregation groups by p.from_date, so those three
-- corrections count as April money. That has two consequences:
--
--   1. The month view ("what did we receive in April?") and the year view
--      ("was January funded correctly?") cannot be reconciled. Neither is
--      wrong about its own question -- January's correction is simply filed
--      under April, and no arithmetic downstream can recover it.
--   2. A correction can be attributed outside the queried range entirely: a
--      January correction that arrived in April is invisible to a
--      January-March query, and a December correction from last year shows up
--      inside a this-year one.
--
-- NULL on purpose, and deliberately NOT backfilled to p.from_date. Rows
-- imported before this migration have no month to recover, and defaulting them
-- to the arrival month would assert precisely the claim that is wrong -- that
-- a correction belongs to the month it arrived in -- while making it
-- indistinguishable from a row where we actually know. NULL means UNKNOWN; the
-- read path falls back to p.from_date via COALESCE, so pre-existing data
-- behaves exactly as it does today and newly imported data gets it right.
--
-- DATE, not TIMESTAMP: this is a calendar month (always the first of it), not
-- an instant, and it never participates in a timezone conversion.

ALTER TABLE government_funding_bill_payments
    ADD COLUMN IF NOT EXISTS billing_month DATE;

COMMENT ON COLUMN government_funding_bill_payments.billing_month IS
    'Month this row applies to (first of month), from the ISBJ "Monat/ Typ" column. NULL = unknown; read path falls back to the bill period from_date.';

-- Aggregations group by the attribution month; without an index they fall back
-- to a sequential scan over every payment row in the org.
CREATE INDEX IF NOT EXISTS idx_bill_payments_billing_month
    ON government_funding_bill_payments (billing_month);
