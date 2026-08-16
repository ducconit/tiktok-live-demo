-- +goose Up
-- Khôi phục id role về slug gốc (admin/system_admin/user) cho DB dev đã bị
-- migration 00010 đảo qua TestMigrate_UpDownStatus (down/up không nhất quán
-- với data cũ). Map theo name (duy nhất) — chính xác với mọi trạng thái data.
-- KHÔNG dùng temp table (sqlc đọc schema từ migrations → sinh model rác).

ALTER TABLE user_roles DROP CONSTRAINT user_roles_role_id_fkey;
ALTER TABLE role_permissions DROP CONSTRAINT role_permissions_role_id_fkey;

-- Đồng bộ bảng tham chiếu theo name (roles.id chưa đổi — vẫn uuid text)
UPDATE user_roles ur SET role_id = 'system_admin' FROM roles r
    WHERE r.name = 'Hệ thống quản trị' AND ur.role_id::text = r.id::text;
UPDATE user_roles ur SET role_id = 'admin' FROM roles r
    WHERE r.name = 'Admin' AND ur.role_id::text = r.id::text;
UPDATE user_roles ur SET role_id = 'user' FROM roles r
    WHERE r.name = 'User' AND ur.role_id::text = r.id::text;
UPDATE role_permissions rp SET role_id = 'system_admin' FROM roles r
    WHERE r.name = 'Hệ thống quản trị' AND rp.role_id::text = r.id::text;
UPDATE role_permissions rp SET role_id = 'admin' FROM roles r
    WHERE r.name = 'Admin' AND rp.role_id::text = r.id::text;
UPDATE role_permissions rp SET role_id = 'user' FROM roles r
    WHERE r.name = 'User' AND rp.role_id::text = r.id::text;

-- Đổi id role theo name (idempotent)
UPDATE roles SET id = 'system_admin' WHERE name = 'Hệ thống quản trị' AND id <> 'system_admin';
UPDATE roles SET id = 'admin' WHERE name = 'Admin' AND id <> 'admin';
UPDATE roles SET id = 'user' WHERE name = 'User' AND id <> 'user';

ALTER TABLE user_roles ADD CONSTRAINT user_roles_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE;
ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE;

-- +goose Down
-- Không có down (chỉ sửa data — ổn định ở up)
SELECT 1;
