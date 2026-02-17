ALTER TABLE government_funding_properties ADD COLUMN label VARCHAR(255) NOT NULL DEFAULT '';

-- Backfill: title-case the value for existing rows
UPDATE government_funding_properties SET label = INITCAP(value) WHERE label = '';

-- Remove the default once backfilled
ALTER TABLE government_funding_properties ALTER COLUMN label DROP DEFAULT;
