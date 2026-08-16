-- +goose Up
-- Reference data: roles + permissions (admin account được seed qua CLI `seed`,
-- vì cần bcrypt hash runtime — không hardcode hash trong SQL).
-- Lưu ý: cột is_system được thêm ở 00009 (system role immutable) — không insert ở đây.

INSERT INTO roles (slug, name, description) VALUES
    ('system_admin', 'Hệ thống quản trị', 'Role hệ thống — toàn quyền, immutable (không xoá/sửa quyền, chỉ đổi tên)'),
    ('admin', 'Admin', 'Quản trị vận hành'),
    ('user', 'User', 'Người dùng thường')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO permissions (slug, name, description) VALUES
    ('dashboard.view', 'Xem dashboard', 'Xem trang tổng quan'),
    ('users.read', 'Xem người dùng', 'Danh sách và chi tiết người dùng'),
    ('users.write', 'Quản lý người dùng', 'Tạo/sửa/xoá người dùng'),
    ('roles.read', 'Xem vai trò', 'Danh sách vai trò và quyền'),
    ('roles.write', 'Quản lý vai trò', 'Tạo/sửa/xoá vai trò, gán quyền'),
    ('settings.read', 'Xem cài đặt', 'Xem cài đặt hệ thống'),
    ('settings.write', 'Sửa cài đặt', 'Sửa cài đặt hệ thống')
ON CONFLICT (slug) DO NOTHING;

-- system_admin: tất cả quyền
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.slug = 'system_admin'
ON CONFLICT DO NOTHING;

-- admin: mọi thứ trừ roles.write
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.slug IN
    ('dashboard.view', 'users.read', 'users.write', 'roles.read', 'settings.read')
WHERE r.slug = 'admin'
ON CONFLICT DO NOTHING;

-- user: chỉ xem dashboard
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.slug = 'dashboard.view'
WHERE r.slug = 'user'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions;
DELETE FROM permissions;
DELETE FROM roles;
