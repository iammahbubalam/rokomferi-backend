-- name: GetContentByKey :one
SELECT * FROM content_blocks
WHERE section_key = $1 LIMIT 1;

-- name: UpsertContent :one
INSERT INTO content_blocks (section_key, content, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (section_key)
DO UPDATE SET content = $2, updated_at = NOW()
RETURNING *;
