-- Add indexes for Product KPIs optimization
CREATE INDEX IF NOT EXISTS idx_products_stock ON products(stock);
CREATE INDEX IF NOT EXISTS idx_products_stock_alert ON products(stock, low_stock_threshold);
CREATE INDEX IF NOT EXISTS idx_products_is_active ON products(is_active);
