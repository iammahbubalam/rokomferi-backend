DROP INDEX IF EXISTS idx_products_tags;
DROP INDEX IF EXISTS idx_products_brand;
ALTER TABLE products
    DROP COLUMN warranty_info,
    DROP COLUMN tags,
    DROP COLUMN brand;

DROP INDEX IF EXISTS idx_variants_attributes;
ALTER TABLE variants
    DROP COLUMN barcode,
    DROP COLUMN dimensions,
    DROP COLUMN weight,
    DROP COLUMN images,
    DROP COLUMN sale_price,
    DROP COLUMN price,
    DROP COLUMN attributes;
