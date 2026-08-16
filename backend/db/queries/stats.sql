-- ============ Stats cho admin dashboard ============

-- name: CountUsersTotal :one
SELECT COUNT(*) FROM users;

-- name: CountUsersActive :one
SELECT COUNT(*) FROM users WHERE is_active = TRUE;

-- name: CountUsersRecent :many
SELECT date_trunc('day', created_at)::date AS day, COUNT(*) AS count
FROM users
WHERE created_at >= now() - interval '30 days'
GROUP BY day
ORDER BY day;

-- name: RoleDistribution :many
SELECT r.id, COUNT(ur.user_id) AS count
FROM roles r
LEFT JOIN user_roles ur ON ur.role_id = r.id
GROUP BY r.id
ORDER BY count DESC;
