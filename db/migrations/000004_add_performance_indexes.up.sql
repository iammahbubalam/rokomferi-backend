-- Composite index for Home Page (Featured Products) query
CREATE INDEX IF NOT EXISTS idx_products_featured_active_created 
ON products(is_featured, is_active, created_at DESC);

-- Composite index for Category listing (Slug + Active)
CREATE INDEX IF NOT EXISTS idx_products_category_slug_active 
ON products(is_active) INCLUDE (is_featured, created_at);
