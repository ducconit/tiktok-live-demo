-- +goose Up
-- Quyền remote config + cache (dashboard admin) — dùng từ đầu nhưng chưa có trong seed.

INSERT INTO permissions (slug, name, description) VALUES
    ('config.read', 'Xem cấu hình', 'Xem cấu hình động (remote config)'),
    ('config.write', 'Sửa cấu hình', 'Sửa cấu hình động (remote config)'),
    ('cache.read', 'Xem cache', 'Xem thông tin cache'),
    ('cache.delete', 'Xoá cache', 'Xoá toàn bộ cache')
ON CONFLICT (slug) DO NOTHING;

-- system_admin: tất cả quyền (kể cả mới)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.slug = 'system_admin'
  AND p.slug IN ('config.read', 'config.write', 'cache.read', 'cache.delete')
ON CONFLICT DO NOTHING;

-- admin: đọc/ghi config + xoá cache
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.slug IN
    ('config.read', 'config.write', 'cache.read', 'cache.delete')
WHERE r.slug = 'admin'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE slug IN ('config.read', 'config.write', 'cache.read', 'cache.delete')
);
DELETE FROM permissions WHERE slug IN ('config.read', 'config.write', 'cache.read', 'cache.delete');
