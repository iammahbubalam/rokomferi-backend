-- Clean up duplicates before adding constraint
DELETE FROM cart_items a USING cart_items b
WHERE a.id < b.id 
  AND a.cart_id = b.cart_id 
  AND a.product_id = b.product_id 
  AND (a.variant_id = b.variant_id OR (a.variant_id IS NULL AND b.variant_id IS NULL));

-- Add Unique Index (Handling NULL variant_id)
CREATE UNIQUE INDEX IF NOT EXISTS idx_cart_items_unique 
ON cart_items (cart_id, product_id, (COALESCE(variant_id, '00000000-0000-0000-0000-000000000000'::uuid)));
