package server

import (
	"net/http"
	"sort"
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/gin-gonic/gin"
)

// ============================================================
// OpenAPI spec TỰ ĐỘNG — generate runtime từ gin routes thật.
//
// Không cần comment annotation (khác swaggo): spec luôn đồng bộ
// 100% với routes đã đăng ký — không bao giờ lệch. Đủ để:
//   - API explorer (Swagger UI, Insomnia, Postman import)
//   - Generate client sơ bộ (openapi-generator, fern...)
//
// Endpoint: GET /api/v1/openapi.json (public — spec không lộ secret)
// ============================================================

const openAPIVersion = "3.0.3"

// openapiDocument — spec tối giản: paths từ routes + components chuẩn.
type openapiDocument struct {
	OpenAPI    string                     `json:"openapi"`
	Info       openapiInfo                `json:"info"`
	Servers    []openapiServer            `json:"servers,omitempty"`
	Paths      map[string]openapiPathItem `json:"paths"`
	Components openapiComponents          `json:"components"`
}

type openapiInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type openapiServer struct {
	URL string `json:"url"`
}

type openapiPathItem map[string]openapiOperation

type openapiOperation struct {
	Tags        []string                   `json:"tags,omitempty"`
	Summary     string                     `json:"summary,omitempty"`
	OperationID string                     `json:"operationId,omitempty"`
	Parameters  []openapiParameter         `json:"parameters,omitempty"`
	Responses   map[string]openapiResponse `json:"responses"`
	Security    []map[string][]string      `json:"security,omitempty"`
}

type openapiParameter struct {
	Name        string        `json:"name"`
	In          string        `json:"in"`
	Required    bool          `json:"required,omitempty"`
	Schema      openapiSchema `json:"schema"`
	Description string        `json:"description,omitempty"`
}

type openapiSchema struct {
	Type string `json:"type,omitempty"`
	Ref  string `json:"$ref,omitempty"`
}

type openapiResponse struct {
	Description string         `json:"description"`
	Content     map[string]any `json:"content,omitempty"`
}

type openapiComponents struct {
	Schemas         map[string]any              `json:"schemas"`
	SecuritySchemes map[string]openapiSecScheme `json:"securitySchemes"`
}

type openapiSecScheme struct {
	Type   string `json:"type"`
	Scheme string `json:"scheme,omitempty"`
	Name   string `json:"name,omitempty"`
	In     string `json:"in,omitempty"`
}

// openapiHandler — GET /api/v1/openapi.json: spec tự generate từ routes.
type openapiHandler struct {
	appName  string
	appTitle string
	version  string
	routes   []string // snapshot: "METHOD /path" (lấy lúc khởi tạo, sau setupRoutes)
	basePath string   // vd "/api/v1"
}

func newOpenAPIHandler(appName, appTitle, version string) *openapiHandler {
	return &openapiHandler{appName: appName, appTitle: appTitle, version: version, basePath: "/api/v1"}
}

// snapshotRoutes — lấy routes thật từ gin engine (gọi sau setupRoutes).
func (h *openapiHandler) snapshotRoutes(engine *gin.Engine) {
	for _, r := range engine.Routes() {
		if r.Path == "/api/v1/openapi.json" {
			continue
		}
		h.routes = append(h.routes, r.Method+" "+r.Path)
	}
	sort.Strings(h.routes)
}

func (h *openapiHandler) get(c *gin.Context) {
	response.OK(c, h.build())
}

// build — dựng document từ snapshot routes.
func (h *openapiHandler) build() openapiDocument {
	doc := openapiDocument{
		OpenAPI: openAPIVersion,
		Info: openapiInfo{
			Title:       h.appTitle,
			Description: "API spec tự động từ routes — " + h.appName,
			Version:     h.version,
		},
		Servers: []openapiServer{{URL: h.basePath}},
		Paths:   map[string]openapiPathItem{},
		Components: openapiComponents{
			Schemas:         buildSchemas(),
			SecuritySchemes: buildSecuritySchemes(),
		},
	}

	for _, route := range h.routes {
		method, path, _ := strings.Cut(route, " ")
		if !strings.HasPrefix(path, h.basePath) {
			continue // chỉ expose API, bỏ route khác (vd /openapi.json nếu có)
		}
		op := h.operation(method, path)
		rel := strings.TrimPrefix(path, h.basePath)
		if rel == "" {
			rel = "/"
		}
		if doc.Paths[rel] == nil {
			doc.Paths[rel] = openapiPathItem{}
		}
		doc.Paths[rel][strings.ToLower(method)] = op
	}
	return doc
}

// operation — mô tả 1 operation từ method + path (không lộ handler nội bộ).
func (h *openapiHandler) operation(method, path string) openapiOperation {
	op := openapiOperation{
		Summary: describeRoute(method, path),
		Responses: map[string]openapiResponse{
			"200": {Description: "Thành công — envelope {code, msg, data, meta}"},
			"400": {Description: "Bad request"},
			"401": {Description: "Thiếu/sai token hoặc API key"},
			"403": {Description: "Không có quyền"},
			"404": {Description: "Không tìm thấy"},
			"422": {Description: "Validation — meta = {field: message}"},
			"500": {Description: "Lỗi nội bộ — msg mặc định"},
			"503": {Description: "Maintenance mode"},
		},
	}

	// Path params → parameters
	for _, seg := range strings.Split(path, "/") {
		if strings.HasPrefix(seg, ":") {
			name := strings.TrimPrefix(seg, ":")
			op.Parameters = append(op.Parameters, openapiParameter{
				Name:     name,
				In:       "path",
				Required: true,
				Schema:   openapiSchema{Type: "string"},
			})
		}
	}

	// Security theo namespace
	switch {
	case strings.Contains(path, "/admin/"):
		op.Security = []map[string][]string{{"bearerAuth": {}}}
		op.Tags = []string{"admin"}
	case strings.Contains(path, "/integrations/"):
		op.Security = []map[string][]string{{"apiKeyAuth": {}}}
		op.Tags = []string{"integrations"}
	case strings.Contains(path, "/public/") && !isPublicFree(path):
		op.Security = []map[string][]string{{"bearerAuth": {}}}
		op.Tags = []string{"public"}
	default:
		op.Tags = []string{"public"}
	}

	op.OperationID = operationID(method, path)
	return op
}

// isPublicFree — endpoint public KHÔNG cần token (login, refresh, register, config...).
func isPublicFree(path string) bool {
	free := []string{
		"/public/auth/login", "/public/auth/refresh", "/public/auth/register",
		"/public/auth/verify-account", "/public/auth/resend-otp",
		"/public/auth/forgot-password", "/public/auth/reset-password",
		"/public/config", "/openapi.json",
	}
	for _, f := range free {
		if strings.HasSuffix(path, f) {
			return true
		}
	}
	return false
}

// describeRoute — mô tả ngắn gọn từ method + path.
func describeRoute(method, path string) string {
	verb := map[string]string{
		http.MethodGet: "Lấy", http.MethodPost: "Tạo", http.MethodPut: "Cập nhật",
		http.MethodPatch: "Cập nhật một phần", http.MethodDelete: "Xoá",
	}[method]
	if verb == "" {
		verb = method
	}
	return verb + " /" + strings.TrimPrefix(path, "/api/v1/")
}

// operationID — chuẩn camelCase: GET /admin/users/:id → getAdminUsersId.
func operationID(method, path string) string {
	rel := strings.TrimPrefix(path, "/api/v1/")
	var sb strings.Builder
	sb.WriteString(strings.ToLower(method))
	for _, p := range strings.Split(rel, "/") {
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, ":") {
			sb.WriteString(strings.ToUpper(p[1:2]))
			if len(p) > 2 {
				sb.WriteString(p[2:])
			}
			continue
		}
		// camel hóa từng từ (api-keys → ApiKeys)
		for _, part := range strings.Split(p, "-") {
			if part == "" {
				continue
			}
			sb.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				sb.WriteString(part[1:])
			}
		}
	}
	return sb.String()
}

// buildSchemas — components.schemas chuẩn của skeleton.
func buildSchemas() map[string]any {
	env := func(required []string, props map[string]any) map[string]any {
		return map[string]any{"type": "object", "required": required, "properties": props}
	}
	return map[string]any{
		"Envelope": env(nil, map[string]any{
			"code": map[string]any{"type": "string", "description": `"0" = thành công; lỗi = HTTP status ("400".."500")`},
			"msg":  map[string]any{"type": "string"},
			"data": map[string]any{"nullable": true},
			"meta": map[string]any{"nullable": true, "description": "phân trang {limit,page,total} | validation {field: message}"},
		}),
		"User": env([]string{"id", "email"}, map[string]any{
			"id":            map[string]any{"type": "string", "format": "uuid"},
			"email":         map[string]any{"type": "string", "format": "email"},
			"full_name":     map[string]any{"type": "string"},
			"avatar_url":    map[string]any{"type": "string"},
			"is_active":     map[string]any{"type": "boolean"},
			"last_login_at": map[string]any{"type": "string", "format": "date-time", "nullable": true},
			"created_at":    map[string]any{"type": "string", "format": "date-time"},
		}),
		"ApiKey": env([]string{"id", "name", "key_prefix"}, map[string]any{
			"id":         map[string]any{"type": "string", "format": "uuid"},
			"name":       map[string]any{"type": "string"},
			"key_prefix": map[string]any{"type": "string", "description": "vd gvs_development_ab12... — KHÔNG phải key đầy đủ"},
			"scopes":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"expires_at": map[string]any{"type": "string", "format": "date-time", "nullable": true},
			"is_active":  map[string]any{"type": "boolean"},
			"revoked_at": map[string]any{"type": "string", "format": "date-time", "nullable": true},
		}),
		"ApiKeyCreated": env([]string{"key"}, map[string]any{
			"key": map[string]any{"type": "string", "description": "Plaintext — chỉ hiển thị ĐÚNG 1 lần lúc tạo/rotate"},
		}),
		"Role": env([]string{"id", "slug", "name"}, map[string]any{
			"id":          map[string]any{"type": "string", "format": "uuid"},
			"slug":        map[string]any{"type": "string"},
			"name":        map[string]any{"type": "string"},
			"description": map[string]any{"type": "string"},
		}),
		"Permission": env([]string{"id", "slug", "name"}, map[string]any{
			"id":   map[string]any{"type": "string", "format": "uuid"},
			"slug": map[string]any{"type": "string"},
			"name": map[string]any{"type": "string"},
		}),
		"Error": env([]string{"code", "msg"}, map[string]any{
			"code": map[string]any{"type": "string", "description": "HTTP status dạng string"},
			"msg":  map[string]any{"type": "string"},
			"data": map[string]any{"nullable": true},
			"meta": map[string]any{"nullable": true, "description": "validation: {field: message}"},
		}),
	}
}

// buildSecuritySchemes — bearer JWT + API key.
func buildSecuritySchemes() map[string]openapiSecScheme {
	return map[string]openapiSecScheme{
		"bearerAuth": {Type: "http", Scheme: "bearer", In: "header", Name: "Authorization"},
		"apiKeyAuth": {Type: "apiKey", In: "header", Name: "X-API-Key"},
	}
}
