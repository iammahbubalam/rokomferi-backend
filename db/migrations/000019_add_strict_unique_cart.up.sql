-- Remove duplicate rows (keeping the one with MAX id)
DELETE FROM cart_items a USING cart_items b
WHERE a.id < b.id
AND a.cart_id = b.cart_id
AND a.product_id = b.product_id
AND a.variant_id = b.variant_id;

-- Drop old lenient index if exists
DROP INDEX IF EXISTS idx_cart_items_unique;

-- Add Strict Unique Index
CREATE UNIQUE INDEX cart_items_cart_product_variant_key ON cart_items (cart_id, product_id, variant_id);
