-- +goose Up
-- ============================================================
-- 1. Role ID: UUID → VARCHAR(64) — admin tự điền (vd system_admin,
--    editor, moderator). id chính là slug (bỏ cột slug riêng).
-- ============================================================

-- Mapping role_id (uuid) → slug trước khi đổi type (cần join roles cũ)
ALTER TABLE user_roles ADD COLUMN role_slug VARCHAR(64);
UPDATE user_roles ur SET role_slug = r.slug FROM roles r WHERE r.id = ur.role_id;
ALTER TABLE role_permissions ADD COLUMN role_slug VARCHAR(64);
UPDATE role_permissions rp SET role_slug = r.slug FROM roles r WHERE r.id = rp.role_id;

-- Bỏ FK tạm (đổi type role_id không được khi FK còn trỏ roles.id uuid)
ALTER TABLE user_roles DROP CONSTRAINT user_roles_role_id_fkey;
ALTER TABLE role_permissions DROP CONSTRAINT role_permissions_role_id_fkey;

-- roles.id → varchar; id = slug cũ (dữ liệu có nghĩa sẵn); bỏ cột slug
ALTER TABLE roles ALTER COLUMN id TYPE VARCHAR(64) USING id::text;
UPDATE roles SET id = slug;
ALTER TABLE roles DROP COLUMN slug;

-- Bảng tham chiếu → role_id varchar
ALTER TABLE user_roles ALTER COLUMN role_id TYPE VARCHAR(64) USING role_slug;
ALTER TABLE user_roles DROP COLUMN role_slug;
ALTER TABLE role_permissions ALTER COLUMN role_id TYPE VARCHAR(64) USING role_slug;
ALTER TABLE role_permissions DROP COLUMN role_slug;

-- Gắn lại FK (role_id varchar → roles.id varchar)
ALTER TABLE user_roles ADD CONSTRAINT user_roles_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE;
ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE;

-- ============================================================
-- 2. UUIDv7 cho mọi bảng có id UUID (PG18 có hàm uuidv7() —
--    time-ordered, tốt cho index B-tree + sort by created).
-- ============================================================
ALTER TABLE users ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE permissions ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE refresh_tokens ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE api_keys ALTER COLUMN id SET DEFAULT uuidv7();

-- +goose Down
-- UUIDv7 → uuid v4 (gen_random_uuid)
ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE permissions ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE refresh_tokens ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE api_keys ALTER COLUMN id SET DEFAULT gen_random_uuid();

-- Role id: varchar → uuid (id hiện tại = slug — không phải uuid — không
-- recover được uuid gốc; down chỉ đổi type theo id text, data giữ nguyên)
ALTER TABLE user_roles DROP CONSTRAINT user_roles_role_id_fkey;
ALTER TABLE role_permissions DROP CONSTRAINT role_permissions_role_id_fkey;
ALTER TABLE user_roles ALTER COLUMN role_id TYPE UUID USING md5(role_id)::uuid;
ALTER TABLE role_permissions ALTER COLUMN role_id TYPE UUID USING md5(role_id)::uuid;
ALTER TABLE roles ALTER COLUMN id TYPE UUID USING md5(id)::uuid;
ALTER TABLE roles ADD COLUMN slug TEXT;
UPDATE roles SET slug = id::text;
ALTER TABLE user_roles ADD CONSTRAINT user_roles_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE;
ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE;
