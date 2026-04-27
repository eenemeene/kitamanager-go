-- 000016: harden budget_item_entries with overlap + value constraints
--
-- Three changes, all DB-level enforcement of invariants the
-- application layer already tries to enforce. Belt-and-suspenders:
-- a service-layer check has a TOCTOU window (two concurrent INSERTs
-- can both pass a SELECT-then-INSERT validation), and direct SQL
-- (admin scripts, future migrations, DBA console) bypasses the app
-- entirely. The DB is the truthful gate.
--
-- 1. Overlap exclusion. service.ValidateNoOverlap reads the table,
--    checks for overlap, then INSERTs in a separate statement. Under
--    READ COMMITTED isolation the check doesn't see another tx's
--    uncommitted INSERT, so two concurrent CreateEntry calls can
--    both pass and both persist overlapping entries. The user later
--    sees two "active" entries on the same date and a non-deterministic
--    "first match wins" in financials.
--
--    The GiST exclusion constraint enforces "no two entries for the
--    same budget item have overlapping date ranges" atomically.
--
--    Modelling the inclusive [from, to] semantics in daterange (which
--    is half-open [from, to) by default) requires `to + 1 day` so
--    a `from..to` of `2025-01-01..2025-12-31` becomes
--    `[2025-01-01, 2026-01-01)`. NULL `to_date` (ongoing) becomes
--    `'infinity'::date`.
--
-- 2+3. CHECK constraints on category and amount_cents. The service
--    layer rejects bad values, but a direct SQL UPDATE would set
--    `category = 'foo'` or `amount_cents = -1` without complaint.
--    The CHECKs make it impossible.

CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE budget_item_entries
    ADD CONSTRAINT budget_item_entries_no_overlap
    EXCLUDE USING gist (
        budget_item_id WITH =,
        daterange(from_date, COALESCE(to_date + 1, 'infinity'::date), '[)') WITH &&
    );

ALTER TABLE budget_items
    ADD CONSTRAINT budget_items_category_valid
    CHECK (category IN ('income', 'expense'));

ALTER TABLE budget_item_entries
    ADD CONSTRAINT budget_item_entries_amount_nonneg
    CHECK (amount_cents >= 0);
