DROP INDEX IF EXISTS idx_users_email_trgm;
DROP INDEX IF EXISTS idx_orders_id_trgm;
DROP INDEX IF EXISTS idx_orders_payment_phone_trgm;
DROP INDEX IF EXISTS idx_orders_payment_trx_trgm;
DROP INDEX IF EXISTS idx_orders_status_created_desc;
DROP INDEX IF EXISTS idx_orders_payment_status;
DROP EXTENSION IF EXISTS pg_trgm;
