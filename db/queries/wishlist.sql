-- name: CreateWishlist :one
INSERT INTO wishlists (user_id) VALUES ($1) RETURNING *;

-- name: GetWishlistByUserID :one
SELECT * FROM wishlists WHERE user_id = $1;

-- name: AddWishlistItem :exec
INSERT INTO wishlist_items (wishlist_id, product_id) 
VALUES ($1, $2)
ON CONFLICT (wishlist_id, product_id) DO NOTHING;

-- name: RemoveWishlistItem :exec
DELETE FROM wishlist_items 
WHERE wishlist_id = $1 AND product_id = $2;

-- name: GetWishlistItems :many
SELECT 
    wi.id as wishlist_item_id,
    wi.product_id,
    wi.created_at as added_at,
    p.name,
    p.slug,
    p.base_price,
    p.sale_price,
    p.media,
    p.stock,
    p.stock_status
FROM wishlist_items wi
JOIN products p ON wi.product_id = p.id
WHERE wi.wishlist_id = $1
ORDER BY wi.created_at DESC;

-- name: CheckItemInWishlist :one
SELECT EXISTS(
    SELECT 1 FROM wishlist_items 
    WHERE wishlist_id = $1 AND product_id = $2
);
