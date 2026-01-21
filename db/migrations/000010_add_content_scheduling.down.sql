-- Revert scheduling fields
DROP INDEX IF EXISTS idx_content_blocks_active;

ALTER TABLE content_blocks DROP COLUMN IF EXISTS end_at;
ALTER TABLE content_blocks DROP COLUMN IF EXISTS start_at;
ALTER TABLE content_blocks DROP COLUMN IF EXISTS is_active;
