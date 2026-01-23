-- Reverse Phase 6: Operational Scale (Inventory SSOT)

-- 1. Restore columns to products table
ALTER TABLE products ADD COLUMN sku VARCHAR(100);
ALTER TABLE products ADD COLUMN stock INTEGER NOT NULL DEFAULT 0;
ALTER TABLE products ADD COLUMN low_stock_threshold INTEGER NOT NULL DEFAULT 5;

-- 2. Restore data from Variants to Products
-- Note: This is lossy for multi-variant products! 
-- We pick the 'Standard' variant or the first one available.
UPDATE products p
SET 
    stock = v.stock,
    sku = v.sku,
    low_stock_threshold = v.low_stock_threshold
FROM variants v
WHERE v.product_id = p.id AND (v.name = 'Standard' OR v.id = (SELECT id FROM variants WHERE product_id = p.id LIMIT 1));

-- 3. Cleanup inventory_logs
UPDATE inventory_logs SET variant_id = NULL WHERE variant_id IN (SELECT id FROM variants WHERE name = 'Standard');

-- 4. Drop the added column from variants
ALTER TABLE variants DROP COLUMN IF EXISTS low_stock_threshold;
