-- +goose Up
-- API keys cho namespace /integrations (server-server auth).
-- Chỉ lưu HASH (SHA-256) của key — plaintext chỉ hiển thị ĐÚNG 1 lần lúc tạo/rotate.
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,          -- SHA-256 hex của key thật
    key_prefix TEXT NOT NULL,               -- vd "gvs_live_ab12..." hiển thị an toàn
    scopes TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,                 -- NULL = không hết hạn
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,
    created_by UUID REFERENCES users (id) ON DELETE SET NULL,
    revoked_at TIMESTAMPTZ,                 -- NULL = còn hiệu lực
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_keys_created_at ON api_keys (created_at);
CREATE INDEX idx_api_keys_hash ON api_keys (key_hash);

-- +goose Down
DROP TABLE IF EXISTS api_keys CASCADE;
