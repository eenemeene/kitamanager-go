-- Reverses 000024: drops the optimistic-concurrency counters. Any in-flight
-- If-Match preconditions stop being enforceable, which is the intended effect of
-- rolling this back.

ALTER TABLE child_contracts DROP COLUMN IF EXISTS version;

ALTER TABLE employee_contracts DROP COLUMN IF EXISTS version;
