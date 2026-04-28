-- 000022: GiST exclusion constraints for contract overlap, both child and employee
--
-- Closes the SELECT-then-INSERT race in
-- internal/store/period.go:ValidateNoOverlap. Under READ COMMITTED isolation
-- two concurrent CreateContract / UpdateContract calls for the same owner can
-- both pass the application-level overlap check (each tx's snapshot doesn't
-- see the other's pending write) and both persist overlapping rows. The
-- billing / funding pipeline then trips over two "active" contracts on the
-- same day for the same child or employee — silent data corruption, hard to
-- forensic-trace once it has happened.
--
-- The pattern mirrors migration 000016 (budget_item_entries). Two differences:
--
-- 1. DEFERRABLE INITIALLY DEFERRED. service.BatchUpdateContracts runs in two
--    phases — phase 1 saves all updates, phase 2 validates overlap against
--    the final state. A swap-pair edit (extend A's To, shift B's From) is
--    transiently overlapping after phase 1's first save and only consistent
--    once phase 2 has reconciled. Defer the check to COMMIT so phase 1 can
--    do its work; phase 2 still runs as a friendly application-level
--    pre-check that returns a clean 409 in the common case. The
--    commit-time 23P01 from this constraint is mapped to 409 by
--    service.mapContractDeferredOverlap when the race actually hits.
--
-- 2. daterange uses `to_date + 1` (rather than `to_date`) because the model's
--    [from, to] semantics are inclusive on both ends, while daterange's
--    canonical form is half-open `[)`. Without the +1, two contracts where
--    A.to == B.from would not be detected as overlapping by &&; they ARE
--    overlapping under our domain rules (see Period.Overlaps in
--    internal/models/period.go). NULL to_date (ongoing) becomes 'infinity'.

CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE child_contracts
    ADD CONSTRAINT child_contracts_no_overlap
    EXCLUDE USING gist (
        child_id WITH =,
        daterange(from_date, COALESCE(to_date + 1, 'infinity'::date), '[)') WITH &&
    ) DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE employee_contracts
    ADD CONSTRAINT employee_contracts_no_overlap
    EXCLUDE USING gist (
        employee_id WITH =,
        daterange(from_date, COALESCE(to_date + 1, 'infinity'::date), '[)') WITH &&
    ) DEFERRABLE INITIALLY DEFERRED;
