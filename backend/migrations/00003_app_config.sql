-- +goose Up
-- Bảng lưu config khi dùng CONFIG_DSN=postgres://...
-- Mỗi dòng = 1 config key (phẳng, có thể chứa dấu chấm: "server.port")
CREATE TABLE app_config (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE app_config;
