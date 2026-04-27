-- 000016 down: drop the constraints from the up migration. Extension
-- stays installed (other tables might come to rely on btree_gist) —
-- safer to leave it than to risk a future migration's silent breakage.

ALTER TABLE budget_item_entries
    DROP CONSTRAINT IF EXISTS budget_item_entries_no_overlap;

ALTER TABLE budget_items
    DROP CONSTRAINT IF EXISTS budget_items_category_valid;

ALTER TABLE budget_item_entries
    DROP CONSTRAINT IF EXISTS budget_item_entries_amount_nonneg;
