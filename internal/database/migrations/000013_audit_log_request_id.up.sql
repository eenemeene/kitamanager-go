-- Adds the correlation id (X-Request-ID / ECS `trace.id`) to every audit
-- row so investigators can group mutations that belong to a single HTTP
-- request together. "20 rows changed" vs "one button click cascaded to
-- 20 changes" is the difference this column makes during incident
-- response — and it lets ops correlate app logs (which already carry
-- X-Request-ID via the structured logger middleware) with audit rows.
--
-- VARCHAR(64) is sized for UUID-like values plus a small margin —
-- long enough for the standard 36-char UUID representation, short
-- enough to keep the index small. Nullable on purpose:
--   - historical rows keep NULL so the migration is safe on live
--     data;
--   - non-HTTP writers (seed imports, background jobs, CLI tooling)
--     legitimately have no request to correlate to and also produce
--     NULL rows.
--
-- Partial index: NULL rows are skipped to keep the index tight. The
-- dominant query is "fetch every row for this request id", which
-- never matches NULL anyway.

ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS request_id VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id
    ON audit_logs(request_id)
    WHERE request_id IS NOT NULL;
