-- The months a manual Korrekturbuchung settles, and why.
--
-- correction_booking already holds the Abrechnungsübersicht's "Korrekturbuchung
-- manuell": a district-level booking the Senate makes by hand, with no child
-- rows behind it. The amount alone is misleading without its months. In the
-- sample Kita year every such booking was a Nachfinanzierung I-Status settling
-- months one or two Kita years before the bill that paid it, so crediting the
-- amount to its arrival year overstated that year by the whole of it.
--
-- Three columns rather than a table of the individual Korrekturübersicht rows.
-- Nobody decides anything per row; the question the figure raises is "which
-- months is this about", and a span answers it. Should per-row detail ever be
-- wanted, it can be added without moving these.
--
-- Nullable, and null is the normal state: nineteen of twenty-two real bills
-- carried no manual booking at all. Null also covers the case where the sheet
-- was present but its rows did not reconcile with the stated Korrektursumme,
-- where the aggregate is kept and the attribution deliberately is not.
--
-- The reason is a short code, never the Korrekturgrund it came from. That text
-- carries children's names and birthdates, and nothing in this schema is a
-- place for it.
ALTER TABLE government_funding_bill_periods
    ADD COLUMN correction_booking_from_month date,
    ADD COLUMN correction_booking_to_month   date,
    ADD COLUMN correction_booking_reason     varchar(64);

-- A span is either wholly present or wholly absent, and ordered.
ALTER TABLE government_funding_bill_periods
    ADD CONSTRAINT correction_booking_span_complete CHECK (
        (correction_booking_from_month IS NULL) = (correction_booking_to_month IS NULL)
    ),
    ADD CONSTRAINT correction_booking_span_ordered CHECK (
        correction_booking_from_month IS NULL
        OR correction_booking_from_month <= correction_booking_to_month
    );

-- No index: every read of these arrives with the bill row it belongs to, and
-- the reasoning in 000029 about one organization's bills numbering in the low
-- thousands applies here too.
