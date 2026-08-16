-- +goose Up
-- Xác thực email (OTP flow) + permissions mới cho admin operations.
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ NULL;

INSERT INTO permissions (slug, name, description) VALUES
    ('admins.read', 'Xem admin', 'Danh sách và chi tiết tài khoản admin'),
    ('admins.write', 'Quản lý admin', 'Tạo/sửa/xoá tài khoản admin'),
    ('config.write', 'Cập nhật config', 'Cập nhật cấu hình hệ thống từ xa (remote config)'),
    ('cache.delete', 'Xoá cache', 'Xoá toàn bộ cache hệ thống')
ON CONFLICT (slug) DO NOTHING;

-- system_admin: đủ quyền quản trị
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.slug = 'system_admin' AND p.slug IN ('admins.read', 'admins.write', 'config.write', 'cache.delete')
ON CONFLICT DO NOTHING;

-- admin: quản trị nhưng không quản lý admin (admin.read/xoá cache/config cho phép vận hành)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.slug IN ('admins.read', 'config.write', 'cache.delete')
WHERE r.slug = 'admin'
ON CONFLICT DO NOTHING;

-- +goose Down
ALTER TABLE users DROP COLUMN email_verified_at;
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE slug IN ('admins.read', 'admins.write', 'config.write', 'cache.delete'));
DELETE FROM permissions WHERE slug IN ('admins.read', 'admins.write', 'config.write', 'cache.delete');
