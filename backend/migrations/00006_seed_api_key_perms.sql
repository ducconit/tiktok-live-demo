-- +goose Up
-- Quyền quản lý API keys (integrations) — gán cho system_admin + admin.

INSERT INTO permissions (slug, name, description) VALUES
    ('api_keys.read', 'Xem API keys', 'Danh sách và chi tiết API keys'),
    ('api_keys.write', 'Quản lý API keys', 'Tạo/sửa/xoá/rotate API keys')
ON CONFLICT (slug) DO NOTHING;

-- system_admin: tất cả quyền (kể cả mới)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.slug = 'system_admin'
  AND p.slug IN ('api_keys.read', 'api_keys.write')
ON CONFLICT DO NOTHING;

-- admin: đọc + ghi API keys
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.slug IN ('api_keys.read', 'api_keys.write')
WHERE r.slug = 'admin'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE slug IN ('api_keys.read', 'api_keys.write')
);
DELETE FROM permissions WHERE slug IN ('api_keys.read', 'api_keys.write');
