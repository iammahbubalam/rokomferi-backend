-- name: GetVariantsByProductID :many
SELECT * FROM variants WHERE product_id = $1;

-- name: GetVariantByID :one
SELECT * FROM variants WHERE id = $1;

-- name: CreateVariant :one
INSERT INTO variants (product_id, name, stock, sku)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateVariant :one
UPDATE variants SET name = $2, stock = $3, sku = $4 WHERE id = $1 RETURNING *;

-- name: DeleteVariant :exec
DELETE FROM variants WHERE id = $1;

-- name: DeleteVariantsByProductID :exec
DELETE FROM variants WHERE product_id = $1;

-- name: CreateInventoryLog :one
INSERT INTO inventory_logs (product_id, variant_id, change_amount, reason, reference_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetInventoryLogs :many
SELECT * FROM inventory_logs 
WHERE ($1::uuid IS NULL OR product_id = $1)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountInventoryLogs :one
SELECT COUNT(*) FROM inventory_logs WHERE ($1::uuid IS NULL OR product_id = $1);
