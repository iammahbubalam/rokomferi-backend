-- L9 Dashboard/Stats Queries: Fully Parameterized (Zero Hardcoded Values)
-- All date ranges, thresholds, limits controlled by frontend via query params

-- name: GetLowStockProducts :many
-- Products below threshold (parameterized - no hardcoded limit)
SELECT 
  id, name, slug, stock, base_price, sale_price, sku, media, stock_status
FROM products
WHERE stock <= sqlc.arg(threshold)::int 
  AND stock > 0
  AND is_active = true
ORDER BY stock ASC
LIMIT sqlc.arg(limit_count)::int;

-- name: GetTopSellingProducts :many
-- Best-selling products by quantity (parameterized date range and limit)
SELECT 
  p.id, p.name, p.slug, p.sku, p.base_price, p.sale_price, p.media,
  SUM(oi.quantity)::bigint as total_sold,
  SUM(oi.subtotal)::numeric as total_revenue
FROM order_items oi
JOIN products p ON p.id = oi.product_id
JOIN orders o ON o.id = oi.order_id
WHERE o.created_at >= sqlc.arg(start_date)::timestamp
  AND o.created_at <= sqlc.arg(end_date)::timestamp
  AND o.status NOT IN ('cancelled', 'returned')
GROUP BY p.id, p.name, p.slug, p.sku, p.base_price, p.sale_price, p.media
ORDER BY total_sold DESC
LIMIT sqlc.arg(limit_count)::int;

-- name: GetCustomerLTV :many
-- Top customers by lifetime value (parameterized date range and limit)
SELECT 
  u.id, u.first_name, u.last_name, u.email,
  COUNT(o.id)::bigint as total_orders,
  SUM(o.total_amount)::numeric as lifetime_value
FROM users u
JOIN orders o ON o.user_id = u.id
WHERE o.created_at >= sqlc.arg(start_date)::timestamp
  AND o.created_at <= sqlc.arg(end_date)::timestamp
  AND o.status NOT IN ('cancelled', 'returned')
GROUP BY u.id, u.first_name, u.last_name, u.email
ORDER BY lifetime_value DESC
LIMIT sqlc.arg(limit_count)::int;

-- name: GetDailySales :many
-- Revenue aggregation by day with parameterized date range
SELECT 
  DATE(created_at) as date,
  COUNT(*)::int as order_count,
  COALESCE(SUM(total_amount), 0)::numeric as total_revenue,
  COALESCE(AVG(total_amount), 0)::numeric as avg_order_value
FROM orders
WHERE created_at >= sqlc.arg(start_date)::timestamp 
  AND created_at <= sqlc.arg(end_date)::timestamp
  AND status NOT IN ('cancelled', 'returned')
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- name: GetRevenueKPIs :one
-- Key performance indicators for a parameterized date range
SELECT 
  COUNT(*)::bigint as total_orders,
  COALESCE(SUM(total_amount), 0)::numeric as total_revenue,
  COALESCE(AVG(total_amount), 0)::numeric as avg_order_value,
  COUNT(DISTINCT user_id)::bigint as unique_customers
FROM orders
WHERE created_at >= sqlc.arg(start_date)::timestamp
  AND created_at <= sqlc.arg(end_date)::timestamp
  AND status NOT IN ('cancelled', 'returned');

-- name: GetDeadStockProducts :many
-- Products with no sales in X days (parameterized)
SELECT 
  p.id, p.name, p.slug, p.stock, p.base_price, p.sku, p.media,
  p.created_at
FROM products p
WHERE p.id NOT IN (
  SELECT DISTINCT oi.product_id 
  FROM order_items oi
  JOIN orders o ON oi.order_id = o.id
  WHERE o.created_at >= NOW() - (sqlc.arg(days)::int || ' days')::interval
    AND o.status NOT IN ('cancelled', 'returned')
)
AND p.stock > 0
AND p.is_active = true
ORDER BY p.stock DESC, p.created_at ASC
LIMIT sqlc.arg(limit_count)::int;

-- name: GetCustomerRetention :one
-- New vs Returning customers (parameterized date range)
SELECT 
  COUNT(DISTINCT CASE WHEN order_number = 1 THEN user_id END)::bigint as new_customers,
  COUNT(DISTINCT CASE WHEN order_number > 1 THEN user_id END)::bigint as returning_customers
FROM (
  SELECT 
    user_id,
    ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at) as order_number
  FROM orders
  WHERE created_at >= sqlc.arg(start_date)::timestamp
    AND created_at <= sqlc.arg(end_date)::timestamp
    AND status NOT IN ('cancelled', 'returned')
) subquery;
