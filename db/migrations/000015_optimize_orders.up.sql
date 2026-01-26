-- Enable pg_trgm for fast text search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Add missing index for payment_status filtering
CREATE INDEX IF NOT EXISTS idx_orders_payment_status ON orders(payment_status);

-- Add composite index for common dashboard view (Status + Date)
CREATE INDEX IF NOT EXISTS idx_orders_status_created_desc ON orders(status, created_at DESC);

-- Add indices for Pre-order searching inside JSONB (Transaction ID, Sender Number)
-- Using GIN with trgm_ops for ILIKE '%%' support
CREATE INDEX IF NOT EXISTS idx_orders_payment_trx_trgm ON orders USING gin ((payment_details->>'transaction_id') gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_orders_payment_phone_trgm ON orders USING gin ((payment_details->>'sender_number') gin_trgm_ops);

-- Add index for ID search (casting UUID to text)
CREATE INDEX IF NOT EXISTS idx_orders_id_trgm ON orders USING gin ((id::text) gin_trgm_ops);

-- Add index for search on User Email (requires joining users, but we can't index across tables directly)
-- Optimization: We rely on users table index. PostgreSQL join with index is fast.
-- Check users email index: 000001 has UNIQUE(email) -> implicitly indexed (btree).
-- To support ILIKE on email, we add TRGM index on users(email).
CREATE INDEX IF NOT EXISTS idx_users_email_trgm ON users USING gin (email gin_trgm_ops);
