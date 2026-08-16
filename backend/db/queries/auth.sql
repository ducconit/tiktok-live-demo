-- ============ Refresh tokens ============

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeUserRefreshTokens :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- ============ RBAC ============

-- name: GetUserRoles :many
SELECT r.* FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1
ORDER BY r.created_at;

-- name: GetUserPermissions :many
SELECT DISTINCT p.slug FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = $1;

-- name: AssignUserRole :exec
INSERT INTO user_roles (user_id, role_id)
VALUES ($1, $2)
ON CONFLICT (user_id, role_id) DO NOTHING;

-- name: RemoveUserRole :exec
DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2;

-- ============ Roles ============

-- name: ListRoles :many
SELECT * FROM roles ORDER BY created_at;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1;

-- name: CreateRole :one
INSERT INTO roles (id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateRole :one
UPDATE roles SET name = $2, description = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;

-- name: SetRolePermissions :exec
INSERT INTO role_permissions (role_id, permission_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- name: ListRolePermissions :many
SELECT p.* FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
WHERE rp.role_id = $1
ORDER BY p.slug;

-- ============ Permissions ============

-- name: ListPermissions :many
SELECT * FROM permissions ORDER BY slug;

-- name: GetPermissionBySlug :one
SELECT * FROM permissions WHERE slug = $1;
