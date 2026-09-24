-- Reverses 000027. Child contracts stop being required to carry any property
-- key, so a contract with no care_type is accepted again and reports its child
-- as worth the parent meal deduction alone.

ALTER TABLE government_funding_periods DROP COLUMN IF EXISTS required_keys;
