-- Revert the bcrypt cut-over. Any bcrypt rows that exist would be
-- meaningless under the SHA-256 verify path, so we clear the table
-- the same way the up did; the column type goes back to CHAR(64) to
-- match the original 000010 schema.

DELETE FROM factor_backup_codes;

ALTER TABLE factor_backup_codes ALTER COLUMN code_hash TYPE CHAR(64);
