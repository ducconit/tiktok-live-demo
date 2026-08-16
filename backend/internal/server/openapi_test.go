package server

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationID(t *testing.T) {
	cases := []struct{ in, want string }{
		{"GET /api/v1/admin/users/:id", "getAdminUsersId"},
		{"POST /api/v1/admin/api-keys", "postAdminApiKeys"},
		{"GET /api/v1/integrations/config", "getIntegrationsConfig"},
		{"PUT /api/v1/admin/api-keys/:id/rotate", "putAdminApiKeysIdRotate"},
		{"DELETE /api/v1/public/me/avatar", "deletePublicMeAvatar"},
	}
	for _, c := range cases {
		method, path, _ := strings.Cut(c.in, " ")
		assert.Equal(t, c.want, operationID(method, path), c.in)
	}
}

func TestDescribeRoute(t *testing.T) {
	assert.Equal(t, "Lấy /admin/users", describeRoute("GET", "/api/v1/admin/users"))
	assert.Equal(t, "Tạo /admin/api-keys", describeRoute("POST", "/api/v1/admin/api-keys"))
	assert.Equal(t, "Xoá /admin/cache", describeRoute("DELETE", "/api/v1/admin/cache"))
}

func TestBuildDocument(t *testing.T) {
	h := newOpenAPIHandler("test-app", "Test App", "1.2.3")
	// Mô phỏng routes đã snapshot
	h.routes = []string{
		"GET /api/v1/admin/users/:id",
		"POST /api/v1/admin/api-keys",
		"GET /api/v1/integrations/config",
		"POST /api/v1/public/auth/login",
		"GET /api/v1/public/auth/me",
		"GET /api/v1/openapi.json", // phải bị loại (không snapshot — test loại tay)
	}
	// snapshotRoutes tự loại /openapi.json — test build trực tiếp sau khi lọc
	var filtered []string
	for _, r := range h.routes {
		if !strings.Contains(r, "openapi.json") {
			filtered = append(filtered, r)
		}
	}
	h.routes = filtered

	doc := h.build()
	assert.Equal(t, "3.0.3", doc.OpenAPI)
	assert.Equal(t, "Test App", doc.Info.Title)
	assert.Equal(t, "1.2.3", doc.Info.Version)
	assert.Len(t, doc.Paths, 5)

	// Security theo namespace
	adminOp := doc.Paths["/admin/users/:id"]["get"]
	require.NotNil(t, adminOp)
	assert.Equal(t, []string{"admin"}, adminOp.Tags)
	assert.Equal(t, []map[string][]string{{"bearerAuth": {}}}, adminOp.Security)
	assert.Equal(t, "getAdminUsersId", adminOp.OperationID)
	// Path param
	require.Len(t, adminOp.Parameters, 1)
	assert.Equal(t, "id", adminOp.Parameters[0].Name)

	intgOp := doc.Paths["/integrations/config"]["get"]
	assert.Equal(t, []map[string][]string{{"apiKeyAuth": {}}}, intgOp.Security)

	// login public — không security; /me cần bearer
	loginOp := doc.Paths["/public/auth/login"]["post"]
	assert.Empty(t, loginOp.Security)
	meOp := doc.Paths["/public/auth/me"]["get"]
	assert.Equal(t, []map[string][]string{{"bearerAuth": {}}}, meOp.Security)

	// Components
	require.Contains(t, doc.Components.Schemas, "Envelope")
	require.Contains(t, doc.Components.Schemas, "ApiKeyCreated")
	require.Contains(t, doc.Components.SecuritySchemes, "bearerAuth")
	require.Contains(t, doc.Components.SecuritySchemes, "apiKeyAuth")
}
