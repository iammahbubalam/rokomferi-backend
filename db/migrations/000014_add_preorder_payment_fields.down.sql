-- Revert Pre-Order fields

DROP INDEX IF EXISTS idx_orders_is_preorder;

ALTER TABLE orders
DROP COLUMN is_preorder,
DROP COLUMN payment_details,
DROP COLUMN paid_amount;
