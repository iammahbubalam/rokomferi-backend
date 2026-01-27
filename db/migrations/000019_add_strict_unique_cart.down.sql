DROP INDEX IF EXISTS cart_items_cart_product_variant_key;

-- Restore old index (Migration 000005)
CREATE UNIQUE INDEX IF NOT EXISTS idx_cart_items_unique 
ON cart_items (cart_id, product_id, (COALESCE(variant_id, '00000000-0000-0000-0000-000000000000'::uuid)));
