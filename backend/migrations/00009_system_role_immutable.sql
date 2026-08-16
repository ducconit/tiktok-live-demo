-- +goose Up
-- Role hệ thống: immutable (không xoá, không sửa quyền — chỉ đổi tên).
-- Đổi role mạnh nhất: super_admin → system_admin (slug chuẩn hệ thống).

ALTER TABLE roles ADD COLUMN is_system boolean NOT NULL DEFAULT false;

-- Role hệ thống mặc định: system_admin (đổi từ super_admin — data cũ theo slug mới,
-- role_id của admin seed không đổi nên quyền/gán vẫn nguyên vẹn)
UPDATE roles SET slug = 'system_admin', name = 'Hệ thống quản trị' WHERE slug = 'super_admin';
UPDATE roles SET is_system = true WHERE slug = 'system_admin';

-- +goose Down
UPDATE roles SET slug = 'super_admin', name = 'Super Admin' WHERE slug = 'system_admin';
ALTER TABLE roles DROP COLUMN is_system;
