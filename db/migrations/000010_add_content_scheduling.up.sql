-- L9: Add scheduling fields to content_blocks for banner scheduling
-- is_active: Toggle content visibility
-- start_at: When content becomes active
-- end_at: When content expires

ALTER TABLE content_blocks 
ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE content_blocks 
ADD COLUMN IF NOT EXISTS start_at TIMESTAMPTZ;

ALTER TABLE content_blocks 
ADD COLUMN IF NOT EXISTS end_at TIMESTAMPTZ;

-- Add partial index for efficient active content lookups
CREATE INDEX IF NOT EXISTS idx_content_blocks_active 
ON content_blocks (section_key) 
WHERE is_active = true;

COMMENT ON COLUMN content_blocks.is_active IS 'L9: Master toggle for content visibility';
COMMENT ON COLUMN content_blocks.start_at IS 'L9: Content activation timestamp (null = immediate)';
COMMENT ON COLUMN content_blocks.end_at IS 'L9: Content expiration timestamp (null = never)';
