-- name: GetProducts :many
SELECT * FROM products 
WHERE (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
AND (sqlc.narg('is_featured')::boolean IS NULL OR is_featured = sqlc.narg('is_featured'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountProducts :one
SELECT COUNT(*) FROM products 
WHERE (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
AND (sqlc.narg('is_featured')::boolean IS NULL OR is_featured = sqlc.narg('is_featured'));

-- name: GetProductBySlug :one
SELECT * FROM products WHERE slug = $1;

-- name: GetProductByID :one
SELECT * FROM products WHERE id = $1;

-- name: GetProductBySKU :one
SELECT * FROM products WHERE sku = $1;

-- name: CreateProduct :one
INSERT INTO products (name, slug, sku, description, base_price, sale_price, stock, stock_status, low_stock_threshold, is_featured, is_active, media, attributes, specifications)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET name = $2, slug = $3, description = $4, base_price = $5, sale_price = $6, 
    stock = $7, stock_status = $8, low_stock_threshold = $9, is_featured = $10, 
    is_active = $11, media = $12, attributes = $13, specifications = $14
WHERE id = $1
RETURNING *;

-- name: UpdateProductStatus :exec
UPDATE products SET is_active = $2 WHERE id = $1;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = $1;

-- name: UpdateProductStock :execrows
UPDATE products SET stock = stock + $2 WHERE id = $1 AND stock + $2 >= 0;

-- name: GetProductsWithCategoryFilter :many
SELECT DISTINCT p.* FROM products p
JOIN product_categories pc ON pc.product_id = p.id
JOIN categories c ON c.id = pc.category_id
WHERE c.slug = $1 AND ($2::boolean IS NULL OR p.is_active = $2)
ORDER BY p.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountProductsWithCategoryFilter :one
SELECT COUNT(DISTINCT p.id) FROM products p
JOIN product_categories pc ON pc.product_id = p.id
JOIN categories c ON c.id = pc.category_id
WHERE c.slug = $1 AND ($2::boolean IS NULL OR p.is_active = $2);

-- name: GetProductsWithSearch :many
SELECT * FROM products
WHERE name ILIKE '%' || $1 || '%' AND ($2::boolean IS NULL OR is_active = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountProductsWithSearch :one
SELECT COUNT(*) FROM products
WHERE name ILIKE '%' || $1 || '%' AND ($2::boolean IS NULL OR is_active = $2);

-- name: GetProductsWithPriceRange :many
SELECT * FROM products
WHERE base_price >= $1 AND base_price <= $2 AND ($3::boolean IS NULL OR is_active = $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: AddProductCategory :exec
INSERT INTO product_categories (product_id, category_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveProductCategory :exec
DELETE FROM product_categories WHERE product_id = $1 AND category_id = $2;

-- name: ClearProductCategories :exec
DELETE FROM product_categories WHERE product_id = $1;

-- name: GetCategoryIDsForProduct :many
SELECT category_id FROM product_categories WHERE product_id = $1;

-- name: GetProductsForCollection :many
SELECT p.* FROM products p
JOIN product_collections pc ON pc.product_id = p.id
WHERE pc.collection_id = $1 AND p.is_active = true
ORDER BY p.created_at DESC;
