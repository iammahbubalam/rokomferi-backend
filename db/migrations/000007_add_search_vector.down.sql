DROP TRIGGER IF EXISTS products_search_vector_update ON products;
DROP FUNCTION IF EXISTS products_search_vector_update;
DROP INDEX IF EXISTS idx_products_search_vector;
ALTER TABLE products DROP COLUMN IF EXISTS search_vector;
