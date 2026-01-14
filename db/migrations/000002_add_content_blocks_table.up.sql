CREATE TABLE content_blocks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    section_key VARCHAR(100) NOT NULL UNIQUE,
    content JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_content_blocks_key ON content_blocks(section_key);
