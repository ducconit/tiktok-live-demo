-- +goose Up
-- Idempotency cho namespace /integrations (server-server):
-- client gửi header Idempotency-Key → cùng key + cùng endpoint → trả lại response cũ
-- (không thực thi lại — tránh đăng ký/trừ tiền 2 lần khi mạng lỗi retry).
CREATE TABLE idempotency_keys (
    key TEXT PRIMARY KEY,               -- sha256(method|path|Idempotency-Key)
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    request_hash TEXT NOT NULL DEFAULT '', -- hash body request (tuỳ chọn — verify cùng request)
    response_status INT NOT NULL,
    response_body TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_idempotency_expires ON idempotency_keys (expires_at);

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys CASCADE;
