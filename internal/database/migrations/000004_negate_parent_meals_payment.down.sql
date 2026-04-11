-- Revert: make parent/meals payments positive again.
UPDATE government_funding_properties
SET payment = -payment
WHERE key = 'parent' AND value = 'meals' AND payment < 0;
