  -- Phase 6: Operational Scale (Inventory SSOT)
-- Enforce SKU-centricity to eliminate data drift and reach L9 commerce standards.

-- 1. Add L9 standard columns to variants
ALTER TABLE variants ADD COLUMN IF NOT EXISTS low_stock_threshold INTEGER NOT NULL DEFAULT 5 CHECK (low_stock_threshold >= 0);
ALTER TABLE variants ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE variants ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- 1b. Add trigger for variants updated_at
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_variants_updated_at') THEN
        CREATE TRIGGER update_variants_updated_at BEFORE UPDATE ON variants
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- 2. Create "Standard" Variants for products that have stock/metadata but no variants
-- This ensures the 1-to-N relationship is strictly enforced (N >= 1)
INSERT INTO variants (product_id, name, stock, sku, low_stock_threshold, attributes)
SELECT 
    p.id, 
    'Standard', 
    p.stock, 
    p.sku, 
    p.low_stock_threshold,
    '{}'::jsonb
FROM products p
LEFT JOIN variants v ON v.product_id = p.id
WHERE v.id IS NULL;

-- 3. Backfill inventory_logs
-- Associate orphaned logs with the first available variant of the product
UPDATE inventory_logs il
SET variant_id = v.id
FROM (
    SELECT DISTINCT ON (product_id) id, product_id 
    FROM variants 
    ORDER BY product_id, created_at ASC -- Assuming we want the oldest/first variant
) v
WHERE il.product_id = v.product_id AND il.variant_id IS NULL;

-- 4. Drop redundant columns from products table
ALTER TABLE products DROP COLUMN stock;
ALTER TABLE products DROP COLUMN low_stock_threshold;
-- Also remove SKU from products as it now lives in Variants (SKU = Stock Keeping Unit)
ALTER TABLE products DROP COLUMN sku;
