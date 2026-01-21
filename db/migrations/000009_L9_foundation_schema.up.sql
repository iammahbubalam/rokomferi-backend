-- Enable UUID extension if not exists (already enabled in init, but safe to ensure)
-- Marketing: Coupons Table
CREATE TABLE IF NOT EXISTS coupons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) UNIQUE NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('percentage', 'fixed')),
    value DECIMAL(12, 2) NOT NULL CHECK (value > 0),
    min_spend DECIMAL(12, 2) DEFAULT 0,
    usage_limit INTEGER DEFAULT 0, -- 0 means unlimited
    used_count INTEGER DEFAULT 0,
    start_at TIMESTAMP,
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_coupons_code ON coupons(code);
CREATE INDEX idx_coupons_is_active ON coupons(is_active);
CREATE INDEX idx_coupons_expires_at ON coupons(expires_at);

-- SEO Command Center: Add SEO columns to key entities

-- Products
ALTER TABLE products ADD COLUMN IF NOT EXISTS meta_title VARCHAR(255);
ALTER TABLE products ADD COLUMN IF NOT EXISTS meta_description TEXT;
ALTER TABLE products ADD COLUMN IF NOT EXISTS meta_keywords TEXT;
ALTER TABLE products ADD COLUMN IF NOT EXISTS og_image TEXT;

-- Categories
ALTER TABLE categories ADD COLUMN IF NOT EXISTS og_image TEXT;
ALTER TABLE categories ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE categories ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- meta_title, meta_description, keywords already exist in categories table (checked usage) but verifying standard naming
-- In init_schema: meta_title, meta_description, keywords exist. We just added og_image and timestamps.

CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Collections
ALTER TABLE collections ADD COLUMN IF NOT EXISTS meta_title VARCHAR(255);
ALTER TABLE collections ADD COLUMN IF NOT EXISTS meta_description TEXT;
ALTER TABLE collections ADD COLUMN IF NOT EXISTS meta_keywords TEXT;
ALTER TABLE collections ADD COLUMN IF NOT EXISTS og_image TEXT;

-- Analytics Engine: Daily Sales Stats
CREATE TABLE IF NOT EXISTS daily_sales_stats (
    date DATE PRIMARY KEY,
    total_revenue DECIMAL(15, 2) NOT NULL DEFAULT 0,
    total_orders INTEGER NOT NULL DEFAULT 0,
    total_items_sold INTEGER NOT NULL DEFAULT 0,
    avg_order_value DECIMAL(12, 2) NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- This table will be populated by cron jobs or triggers

-- Banner Scheduling: Enhance content_blocks
-- Since content_blocks is generic JSON, we can enforce some schema validation OR add explicit columns.
-- Adding explicit columns makes querying 'active banners' much faster and cleaner.
ALTER TABLE content_blocks ADD COLUMN IF NOT EXISTS start_at TIMESTAMP;
ALTER TABLE content_blocks ADD COLUMN IF NOT EXISTS end_at TIMESTAMP;
ALTER TABLE content_blocks ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;

CREATE INDEX idx_content_blocks_active_time ON content_blocks(is_active, start_at, end_at);

-- L9 Optimization: Indexes for Sorting and Filtering
CREATE INDEX IF NOT EXISTS idx_products_base_price ON products(base_price);
CREATE INDEX IF NOT EXISTS idx_coupons_created_at ON coupons(created_at DESC);

-- Trigger adjustments for updated_at on new tables
CREATE TRIGGER update_coupons_updated_at BEFORE UPDATE ON coupons
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
