-- Create Refunds table
CREATE TABLE IF NOT EXISTS refunds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    amount DECIMAL(12, 2) NOT NULL CHECK (amount > 0),
    reason TEXT,
    restock_items BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_refunds_order_id ON refunds(order_id);

-- Add refunded_amount to orders for easier tracking
ALTER TABLE orders ADD COLUMN IF NOT EXISTS refunded_amount DECIMAL(12, 2) NOT NULL DEFAULT 0;

-- Trigger to update updated_at is generic for refunds if needed, but refunds are usually immutable events.
-- No trigger needed for refunds table itself unless we allow updates.
