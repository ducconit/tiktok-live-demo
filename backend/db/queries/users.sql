-- name: CreateUser :one
INSERT INTO users (email, password_hash, full_name, avatar_url, is_active)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM users
WHERE (sqlc.arg(q)::text = '' OR email ILIKE '%' || sqlc.arg(q) || '%' OR full_name ILIKE '%' || sqlc.arg(q) || '%')
  AND (sqlc.narg(is_active)::boolean IS NULL OR is_active = sqlc.narg(is_active))
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountUsers :one
SELECT COUNT(*) FROM users
WHERE (sqlc.arg(q)::text = '' OR email ILIKE '%' || sqlc.arg(q) || '%' OR full_name ILIKE '%' || sqlc.arg(q) || '%')
  AND (sqlc.narg(is_active)::boolean IS NULL OR is_active = sqlc.narg(is_active));

-- name: UpdateUser :one
UPDATE users
SET full_name = $2, avatar_url = $3, is_active = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1;

-- name: SetEmailVerified :exec
UPDATE users SET email_verified_at = now(), updated_at = now() WHERE id = $1;

-- name: UpdateProfile :one
UPDATE users SET full_name = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: UpdateAvatar :one
UPDATE users SET avatar_url = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: ListUsersByRole :many
SELECT u.* FROM users u
JOIN user_roles ur ON ur.user_id = u.id
JOIN roles r ON r.id = ur.role_id
WHERE r.slug = ANY(sqlc.arg(slugs)::text[])
ORDER BY u.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountUsersByRole :one
SELECT COUNT(*) FROM users u
JOIN user_roles ur ON ur.user_id = u.id
JOIN roles r ON r.id = ur.role_id
WHERE r.slug = ANY(sqlc.arg(slugs)::text[]);

-- name: UpdateUserLastLogin :exec
UPDATE users SET last_login_at = now() WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
