-- Revert Banner Scheduling
ALTER TABLE content_blocks DROP COLUMN IF EXISTS is_active;
ALTER TABLE content_blocks DROP COLUMN IF EXISTS end_at;
ALTER TABLE content_blocks DROP COLUMN IF EXISTS start_at;

-- Revert Analytics
DROP TABLE IF EXISTS daily_sales_stats;

-- Revert SEO Columns
ALTER TABLE collections DROP COLUMN IF EXISTS og_image;
ALTER TABLE collections DROP COLUMN IF EXISTS meta_keywords;
ALTER TABLE collections DROP COLUMN IF EXISTS meta_description;
ALTER TABLE collections DROP COLUMN IF EXISTS meta_title;

ALTER TABLE categories DROP COLUMN IF EXISTS og_image;
ALTER TABLE categories DROP COLUMN IF EXISTS updated_at;
ALTER TABLE categories DROP COLUMN IF EXISTS created_at;
DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;

ALTER TABLE products DROP COLUMN IF EXISTS og_image;
ALTER TABLE products DROP COLUMN IF EXISTS meta_keywords;
ALTER TABLE products DROP COLUMN IF EXISTS meta_description;
ALTER TABLE products DROP COLUMN IF EXISTS meta_title;

-- Revert Marketing
DROP TABLE IF EXISTS coupons;

-- Drop L9 Indexes
DROP INDEX IF EXISTS idx_products_base_price;
DROP INDEX IF EXISTS idx_coupons_created_at;
