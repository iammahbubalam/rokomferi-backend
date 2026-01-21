-- name: GetCartByUserID :one
SELECT * FROM carts WHERE user_id = $1;

-- name: CreateCart :one
INSERT INTO carts (user_id) VALUES ($1) RETURNING *;

-- name: GetCartItems :many
SELECT ci.*, p.name, p.slug, p.base_price, p.sale_price, p.media, p.stock
FROM cart_items ci
JOIN products p ON p.id = ci.product_id
WHERE ci.cart_id = $1;

-- name: GetCartWithItems :many
SELECT 
    c.id as cart_id,
    c.user_id,
    ci.id as item_id,
    ci.product_id,
    ci.variant_id,
    ci.quantity,
    p.name,
    p.slug,
    p.base_price,
    p.sale_price,
    p.media,
    p.stock
FROM carts c
LEFT JOIN cart_items ci ON c.id = ci.cart_id
LEFT JOIN products p ON ci.product_id = p.id
WHERE c.user_id = $1;


-- name: GetCartItemByProductID :one
SELECT * FROM cart_items WHERE cart_id = $1 AND product_id = $2;

-- name: UpsertCartItemAtomic :many
WITH 
  user_cart AS (
    SELECT id FROM carts WHERE user_id = sqlc.arg(user_id)
  ),
  stock_valid AS (
    SELECT p.id FROM products p
    WHERE p.id = sqlc.arg(product_id)
      AND p.stock >= sqlc.arg(quantity)
      AND (p.stock_status IS NULL OR p.stock_status != 'out_of_stock')
  ),
  upserted AS (
    INSERT INTO cart_items (cart_id, product_id, variant_id, quantity)
    SELECT user_cart.id, sqlc.arg(product_id), sqlc.arg(variant_id), sqlc.arg(quantity)
    FROM user_cart
    CROSS JOIN stock_valid
    ON CONFLICT (cart_id, product_id, (COALESCE(variant_id, '00000000-0000-0000-0000-000000000000'::uuid)))
    DO UPDATE SET quantity = EXCLUDED.quantity
    RETURNING cart_id
  )
SELECT ci.id, ci.cart_id, ci.product_id, ci.variant_id, ci.quantity,
       p.name, p.slug, p.base_price, p.sale_price, p.media, p.stock
FROM cart_items ci
JOIN products p ON p.id = ci.product_id
WHERE ci.cart_id = (SELECT cart_id FROM upserted LIMIT 1);




-- name: AtomicRemoveCartItem :exec
DELETE FROM cart_items ci
USING carts c
WHERE ci.cart_id = c.id
  AND c.user_id = $1
  AND ci.product_id = $2;

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

-- name: HasPurchasedProduct :one
SELECT EXISTS (
    SELECT 1
    FROM order_items oi
    JOIN orders o ON o.id = oi.order_id
    WHERE o.user_id = $1 
      AND oi.product_id = $2
      AND o.status = 'delivered'
);
SELECT EXISTS (
    SELECT 1
    FROM order_items oi
    JOIN orders o ON o.id = oi.order_id
    WHERE o.user_id = $1 
      AND oi.product_id = $2
      AND o.status = 'delivered'
);
