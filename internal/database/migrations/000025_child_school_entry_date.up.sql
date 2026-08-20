-- When a child leaves the Kita for school.
--
-- The date was computed from the birthdate alone: 31 July of the year the child
-- turns six by the Stichtag. That is right for most children and wrong for any
-- child granted a Zurückstellung (Berlin: § 42 Abs. 3 SchulG), who stays in the
-- Kita a further year. Nothing could record that, so the Muss/Kann badge named
-- the wrong year and the contract-overrun warning fired on a correct contract --
-- offering, in one click, to shorten it to the superseded date.
--
-- NULL keeps the computed behaviour, which is why every existing row is correct
-- as it stands and no backfill is needed. A value overrides it.
--
-- Deliberately on `children` rather than the shared person columns: employees do
-- not go to school. It is stored as the date school starts (the Einschulungs-
-- termin, as the market calls it), not as an offset from the computed date --
-- Bayern's Einschulungskorridor and Bremen's Karenzzeit leave the "regular" date
-- undefined for a band of birthdates, so there is nothing to offset from, and a
-- Bayern deferral granted up to 31 January is not a whole number of years away
-- from anything.

ALTER TABLE children
    ADD COLUMN IF NOT EXISTS school_entry_date DATE;
