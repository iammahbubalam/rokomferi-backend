-- Clean up any existing invalid data (orphan simple products in cart)
DELETE FROM cart_items WHERE variant_id IS NULL;

-- Enforce strict variant requirement
ALTER TABLE cart_items 
    ALTER COLUMN variant_id SET NOT NULL;
