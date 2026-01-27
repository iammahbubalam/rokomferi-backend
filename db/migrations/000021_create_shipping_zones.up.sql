CREATE TABLE IF NOT EXISTS shipping_zones (
    id SERIAL PRIMARY KEY,
    key VARCHAR(50) UNIQUE NOT NULL,
    label VARCHAR(100) NOT NULL,
    cost DECIMAL(12, 2) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed initial data
INSERT INTO shipping_zones (key, label, cost) VALUES 
('inside_dhaka', 'Inside Dhaka', 80.00),
('outside_dhaka', 'Outside Dhaka', 150.00)
ON CONFLICT (key) DO NOTHING;
