package user

import (
	"context"
	"os"
	"testing"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/internal/rbac"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPool — pool DB thật từ DATABASE_URL (hoặc .env). Skip khi không có DB
// (CI — case đã biết, chưa kiểm chứng ở CI).
func testPool(t *testing.T) *database.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("thiếu DATABASE_URL — skip integration (case: CreateAdminFromCLI; chưa kiểm chứng ở CI)")
	}
	pool, err := database.NewPool(context.Background(), config.DatabaseConfig{URL: url})
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestCreateAdminFromCLI_Success(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	email := "cli-admin-" + uuid.NewString()[:8] + "@example.com"

	err := CreateAdminFromCLI(ctx, pool, AdminCLIParams{
		Email:    email,
		Password: "admin12345",
		FullName: "CLI Admin",
		Roles:    []string{"admin", "user"},
	})
	require.NoError(t, err)

	// Verify: user tồn tại + verified + đủ 2 role
	u, err := NewRepo(pool).GetByEmail(ctx, email)
	require.NoError(t, err)
	assert.NotNil(t, u.EmailVerifiedAt, "admin tạo qua CLI phải xác thực email ngay")

	roles, err := rbac.NewRepo(pool).GetUserRoles(ctx, u.ID)
	require.NoError(t, err)
	got := map[string]bool{}
	for _, r := range roles {
		got[r.ID] = true
	}
	assert.True(t, got["admin"], "phải có role admin")
	assert.True(t, got["user"], "phải có role user")

	// Login được bằng mật khẩu
	_, err = NewRepo(pool).GetByEmail(ctx, email)
	require.NoError(t, err)
}

func TestCreateAdminFromCLI_EmailTaken(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	email := "cli-admin-" + uuid.NewString()[:8] + "@example.com"

	require.NoError(t, CreateAdminFromCLI(ctx, pool, AdminCLIParams{
		Email: email, Password: "admin12345", Roles: []string{"admin"},
	}))
	// Tạo lại cùng email → lỗi, KHÔNG ghi đè
	err := CreateAdminFromCLI(ctx, pool, AdminCLIParams{
		Email: email, Password: "khac-12345", FullName: "Khác", Roles: []string{"user"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "đã tồn tại")
}

func TestCreateAdminFromCLI_RoleNotFound_Rollback(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	email := "cli-admin-" + uuid.NewString()[:8] + "@example.com"

	// Role "khong-ton-tai" → lỗi → toàn bộ ROLLBACK (không để user mồ côi)
	err := CreateAdminFromCLI(ctx, pool, AdminCLIParams{
		Email: email, Password: "admin12345", Roles: []string{"admin", "khong-ton-tai"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "không tồn tại")

	_, err = NewRepo(pool).GetByEmail(ctx, email)
	require.Error(t, err, "user phải bị rollback khi role không tồn tại")
}

func TestCreateAdminFromCLI_InvalidInput(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	// Email sai format → lỗi trước khi đụng DB
	err := CreateAdminFromCLI(ctx, pool, AdminCLIParams{Email: "not-an-email", Password: "admin12345"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email")

	// Password ngắn → lỗi
	err = CreateAdminFromCLI(ctx, pool, AdminCLIParams{Email: "a@b.com", Password: "short"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tối thiểu 8")

	// Roles rỗng → mặc định "system_admin" (role hệ thống — không lỗi)
	email := "cli-admin-" + uuid.NewString()[:8] + "@example.com"
	err = CreateAdminFromCLI(ctx, pool, AdminCLIParams{Email: email, Password: "admin12345"})
	require.NoError(t, err)
	u, err := NewRepo(pool).GetByEmail(ctx, email)
	require.NoError(t, err)
	roles, _ := rbac.NewRepo(pool).GetUserRoles(ctx, u.ID)
	assert.Len(t, roles, 1)
	assert.Equal(t, "system_admin", roles[0].ID)
}
