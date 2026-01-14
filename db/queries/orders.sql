-- name: GetCartByUserID :one
SELECT * FROM carts WHERE user_id = $1;

-- name: CreateCart :one
INSERT INTO carts (user_id) VALUES ($1) RETURNING *;

-- name: GetCartItems :many
SELECT ci.*, p.name, p.slug, p.base_price, p.sale_price, p.media, p.stock
FROM cart_items ci
JOIN products p ON p.id = ci.product_id
WHERE ci.cart_id = $1;

-- name: GetCartItemByProductID :one
SELECT * FROM cart_items WHERE cart_id = $1 AND product_id = $2;

-- name: AddCartItem :one
INSERT INTO cart_items (cart_id, product_id, variant_id, quantity)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateCartItemQuantity :exec
UPDATE cart_items SET quantity = $2 WHERE id = $1;

-- name: RemoveCartItem :exec
DELETE FROM cart_items WHERE id = $1;

-- name: ClearCart :exec
DELETE FROM cart_items WHERE cart_id = $1;

-- name: CreateOrder :one
INSERT INTO orders (user_id, status, total_amount, shipping_address, payment_method, payment_status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetOrderByID :one
SELECT * FROM orders WHERE id = $1;

-- name: GetOrdersByUserID :many
SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC;

-- name: GetAllOrders :many
SELECT o.*, u.email, u.first_name, u.last_name
FROM orders o
JOIN users u ON u.id = o.user_id
WHERE ($1::text = '' OR o.status = $1)
ORDER BY o.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountOrders :one
SELECT COUNT(*) FROM orders WHERE ($1::text = '' OR status = $1);

-- name: UpdateOrderStatus :exec
UPDATE orders SET status = $2 WHERE id = $1;

-- name: CreateOrderItem :one
INSERT INTO order_items (order_id, product_id, variant_id, quantity, price)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetOrderItems :many
SELECT oi.*, p.name, p.slug, p.media
FROM order_items oi
JOIN products p ON p.id = oi.product_id
WHERE oi.order_id = $1;
