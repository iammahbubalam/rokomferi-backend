-- Add Pre-Order and Partial Payment fields to orders table

ALTER TABLE orders
ADD COLUMN paid_amount NUMERIC(10, 2) NOT NULL DEFAULT 0,
ADD COLUMN payment_details JSONB NOT NULL DEFAULT '{}'::jsonb,
ADD COLUMN is_preorder BOOLEAN NOT NULL DEFAULT FALSE;

-- Add index for fast filtering of preorder items
CREATE INDEX idx_orders_is_preorder ON orders(is_preorder);
