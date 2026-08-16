-- name: CreateAPIKey :one
INSERT INTO api_keys (name, key_hash, key_prefix, scopes, expires_at, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetAPIKeyByID :one
SELECT * FROM api_keys WHERE id = $1;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys WHERE key_hash = $1;

-- name: ListAPIKeys :many
SELECT * FROM api_keys
WHERE (sqlc.arg(q)::text = '' OR name ILIKE '%' || sqlc.arg(q) || '%')
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountAPIKeys :one
SELECT COUNT(*) FROM api_keys
WHERE (sqlc.arg(q)::text = '' OR name ILIKE '%' || sqlc.arg(q) || '%');

-- name: UpdateAPIKey :one
UPDATE api_keys
SET name = $2, scopes = $3, expires_at = $4, is_active = $5, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: RevokeAPIKey :exec
UPDATE api_keys SET is_active = FALSE, revoked_at = now(), updated_at = now() WHERE id = $1;

-- name: TouchAPIKeyLastUsed :exec
UPDATE api_keys SET last_used_at = now() WHERE id = $1;
