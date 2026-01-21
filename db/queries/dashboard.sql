-- name: GetLowStockProducts :many
SELECT id, name, sku, stock, stock_status 
FROM products 
WHERE stock <= low_stock_threshold AND is_active = true
ORDER BY stock ASC
LIMIT 50;

-- name: GetTopSellingProducts :many
SELECT p.id, p.name, p.sku, SUM(oi.quantity) as total_sold, SUM(oi.subtotal) as total_revenue
FROM order_items oi
JOIN products p ON p.id = oi.product_id
JOIN orders o ON o.id = oi.order_id
WHERE o.created_at >= $1
GROUP BY p.id, p.name, p.sku
ORDER BY total_sold DESC
LIMIT 10;

-- name: GetCustomerLTV :many
SELECT u.id, u.first_name, u.last_name, u.email, COUNT(o.id) as total_orders, SUM(o.total_amount) as lifetime_value
FROM users u
JOIN orders o ON o.user_id = u.id
WHERE o.status = 'delivered'
GROUP BY u.id, u.first_name, u.last_name, u.email
ORDER BY lifetime_value DESC
LIMIT 50;
