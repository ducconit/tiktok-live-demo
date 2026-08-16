-- name: GetIdempotency :one
SELECT * FROM idempotency_keys WHERE key = $1 AND expires_at > now();

-- name: InsertIdempotency :one
INSERT INTO idempotency_keys (key, method, path, request_hash, response_status, response_body, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (key) DO NOTHING
RETURNING *;

-- name: DeleteExpiredIdempotency :exec
DELETE FROM idempotency_keys WHERE expires_at <= now();
