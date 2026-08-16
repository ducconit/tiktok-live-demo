// Package i18n — đa ngôn ngữ (go-i18n v2, chuẩn Go ecosystem).
//
// Message trong files (messages/*.json) — sửa KHÔNG cần rebuild (embed.FS).
// Thêm ngôn ngữ/key: sửa vi.json (nguồn) → chạy `make i18n:merge` → dịch en.json
// → merge lại (chi tiết docs/i18n.md).
//
// Quy tắc: message ID LUÔN literal tĩnh (goi18n extract/merge không bắt được
// ID nối biến) — nội dung động qua {{.Field}} template data.
package i18n

import (
	"embed"
	"encoding/json"
	"sync"

	"github.com/ducconit/tiktok-live-platform/backend/core/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed messages/*.json
var messagesFS embed.FS

// DefaultLang — ngôn ngữ mặc định (zero-config: không set gì → tiếng Việt).
const DefaultLang = "vi"

var (
	bundleOnce sync.Once
	bundle     *i18n.Bundle
)

// Bundle — bundle singleton (thread-safe): vi (default) + en.
func Bundle() *i18n.Bundle {
	bundleOnce.Do(func() {
		b := i18n.NewBundle(language.Vietnamese)
		b.RegisterUnmarshalFunc("json", json.Unmarshal)
		if _, err := b.LoadMessageFileFS(messagesFS, "messages/vi.json"); err != nil {
			panic("i18n: load vi.json: " + err.Error())
		}
		// en.json optional — thiếu key vẫn fallback về vi (bundle chấp nhận file thiếu)
		_, _ = b.LoadMessageFileFS(messagesFS, "messages/en.json")
		bundle = b
	})
	return bundle
}

// T — render message theo lang; fallback: lang → DefaultLang (vi) → trả về chính ID
// (dev thấy ngay key thiếu, không bị nuốt lỗi). TemplateData cho {{.Field}}.
func T(lang, id string, data map[string]any) string {
	if lang == "" {
		lang = DefaultLang
	}
	localizer := i18n.NewLocalizer(Bundle(), lang, DefaultLang)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: data,
	})
	if err != nil {
		// Không tìm thấy ở mọi lang — trả ID (lộ key thiếu cho dev)
		return id
	}
	return msg
}

// Lang — ngôn ngữ của request (set bởi Middleware; mặc định DefaultLang).
func Lang(c *gin.Context) string {
	if v, ok := c.Get(ctxkey.LangKey); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return DefaultLang
}

// Middleware — parse Accept-Language → set "lang" vào context (fallback DefaultLang).
// Đặt SAU requestID — mọi handler/service đọc qua i18n.Lang(c).
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := DefaultLang
		if header := c.GetHeader("Accept-Language"); header != "" {
			tags, _, err := language.ParseAcceptLanguage(header)
			if err == nil && len(tags) > 0 {
				// Ưu tiên: khớp ngôn ngữ có trong bundle (vi/en), fallback default
				for _, t := range tags {
					base := t.String()
					switch {
					case base == "vi" || base == "en":
						lang = base
					case len(base) > 2 && (base[:2] == "vi" || base[:2] == "en"):
						lang = base[:2]
					}
					if lang != DefaultLang || base == "vi" || base == "en" {
						break
					}
				}
			}
		}
		c.Set(ctxkey.LangKey, lang)
		c.Next()
	}
}
