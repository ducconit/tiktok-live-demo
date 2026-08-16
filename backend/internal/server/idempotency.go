package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// ============================================================
// Idempotency — cho namespace /integrations (server-server).
//
// Client gửi header `Idempotency-Key` (POST/PUT/PATCH):
//   - Cùng key + cùng endpoint (method + path) → trả lại response CŨ (replay),
//     KHÔNG thực thi lại — tránh "đăng ký/trừ tiền 2 lần" khi mạng lỗi → client retry.
//   - Chưa có → thực thi, lưu response (status + body) vào bảng idempotency_keys.
//   - Response 5xx KHÔNG lưu (client được phép retry với key mới).
//
// TTL mặc định 24h — sau đó key hết hạn (expires_at), có thể dùng lại.
// ============================================================

const (
	idempotencyTTL     = 24 * time.Hour
	idempotencyKeyName = "Idempotency-Key"
)

// idempotencyHandler — middleware + DB access (db.Queries qua pool).
type idempotencyHandler struct {
	q   *db.Queries
	ttl time.Duration
}

func newIdempotencyHandler(p *database.Pool) *idempotencyHandler {
	return &idempotencyHandler{q: db.New(p.Write()), ttl: idempotencyTTL}
}

// Middleware — áp dụng cho nhóm route (integrations).
func (h *idempotencyHandler) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Chỉ áp dụng cho method có side-effect + có header
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
		default:
			c.Next()
			return
		}
		key := c.GetHeader(idempotencyKeyName)
		if key == "" {
			c.Next()
			return
		}

		idemKey := idemKeyHash(c.Request.Method, c.Request.URL.Path, key)

		// 1) Đã có response → replay (không chạy handler)
		if rec, err := h.q.GetIdempotency(c, idemKey); err == nil {
			c.Data(int(rec.ResponseStatus), "application/json", []byte(rec.ResponseBody))
			c.Abort()
			return
		} else if !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("idempotency: đọc key lỗi — bỏ qua (thực thi bình thường)", "err", err)
		}

		// 2) Chưa có → capture response rồi lưu
		w := &idemResponseWriter{ResponseWriter: c.Writer}
		c.Writer = w
		c.Next()

		// 5xx không lưu — client retry với key khác
		if w.status >= http.StatusInternalServerError {
			return
		}
		if _, err := h.q.InsertIdempotency(c, db.InsertIdempotencyParams{
			Key:            idemKey,
			Method:         c.Request.Method,
			Path:           c.Request.URL.Path,
			RequestHash:    "",
			ResponseStatus: int32(w.status),
			ResponseBody:   w.body.String(),
			ExpiresAt:      time.Now().Add(h.ttl),
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Conflict: request song song cùng key đang chạy — log, không phạt (handler đã chạy)
				slog.Warn("idempotency: trùng key đang xử lý (request song song)", "key", idemKey)
				return
			}
			slog.Error("idempotency: lưu response lỗi", "err", err)
		}
	}
}

// idemKeyHash — khóa idempotency: sha256(method|path|client-key) — scope theo endpoint.
func idemKeyHash(method, path, clientKey string) string {
	sum := sha256.Sum256([]byte(method + "|" + path + "|" + clientKey))
	return hex.EncodeToString(sum[:])
}

// idemResponseWriter — capture status + body (để replay/lưu).
type idemResponseWriter struct {
	gin.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *idemResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *idemResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
