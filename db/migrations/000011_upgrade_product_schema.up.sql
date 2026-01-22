-- Phase 1.5: L9 World-Class Product Management Schema

-- 1. Upgrade variants table for deep structured data & per-variant pricing
ALTER TABLE variants
    ADD COLUMN attributes JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN price NUMERIC(10, 2), -- Nullable: inherits from product if null
    ADD COLUMN sale_price NUMERIC(10, 2),
    ADD COLUMN images TEXT[] DEFAULT '{}', -- Gallery per variant
    ADD COLUMN weight NUMERIC(8, 3), -- Physical weight for shipping
    ADD COLUMN dimensions JSONB, -- {"l": 10, "w": 5, "h": 2}
    ADD COLUMN barcode TEXT;

-- 2. Performance Indexing for JSONB Attributes (L9 Standard)
CREATE INDEX idx_variants_attributes ON variants USING GIN (attributes);

-- 3. Upgrade products table for organization & rich metadata
ALTER TABLE products
    ADD COLUMN brand TEXT,
    ADD COLUMN tags TEXT[] DEFAULT '{}',
    ADD COLUMN warranty_info JSONB;

-- 4. Indexing for fast filtering
CREATE INDEX idx_products_brand ON products(brand);
CREATE INDEX idx_products_tags ON products USING GIN (tags);
