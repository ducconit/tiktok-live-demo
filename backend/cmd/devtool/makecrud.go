package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// makeCrudCmd — "artisan" của skeleton: sinh khung CRUD cho domain mới.
// v1: sinh file mẫu repo/service/handler; queries.sql + migration viết tay rồi sqlc generate.
func makeCrudCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "make:crud <domain>",
		Short: "Sinh khung CRUD cho domain mới (vd: make:crud product)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw := strings.ToLower(args[0])
			if strings.Contains(raw, " ") {
				return fmt.Errorf("domain không được chứa khoảng trắng")
			}
			dir := filepath.Join("internal", raw)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}

			Pascal := toPascal(raw)
			files := map[string]string{
				"repo.go":    strings.ReplaceAll(repoTmpl, "{{Domain}}", Pascal),
				"service.go": strings.ReplaceAll(serviceTmpl, "{{Domain}}", Pascal),
				"handler.go": strings.ReplaceAll(handlerTmpl, "{{Domain}}", Pascal),
			}
			for name, content := range files {
				path := filepath.Join(dir, name)
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("%s đã tồn tại — không ghi đè", path)
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					return err
				}
			}

			fmt.Println("✔ Đã tạo khung CRUD tại internal/" + raw)
			fmt.Println("Bước tiếp theo:")
			fmt.Println("  1. Viết migration:      migrations/0000x_<domain>.sql")
			fmt.Println("  2. Viết queries:        db/queries/" + raw + ".sql")
			fmt.Println("  3. Generate:            sqlc generate")
			fmt.Println("  4. Đăng ký route:       internal/server/routes.go")
			return nil
		},
	}
}

func toPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

const repoTmpl = `package {{Domain}}

import (
	"context"

	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo — wrapper sqlc cho {{Domain}}.
type Repo struct {
	q *db.Queries
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{q: db.New(pool)}
}

// TODO: wrapper các hàm sqlc — viết db/queries/xxx.sql rồi chạy sqlc generate
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (db.{{Domain}}, error) {
	panic("implement me")
}
`

const serviceTmpl = `package {{Domain}}

// Service — nghiệp vụ {{Domain}} (không biết HTTP).
type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// TODO: nghiệp vụ CRUD — tham khảo internal/user/service.go
`

const handlerTmpl = `package {{Domain}}

import (
	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/ducconit/tiktok-live-platform/backend/core/validation"
	"github.com/gin-gonic/gin"
)

// Handler — HTTP layer cho {{Domain}}.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes — gắn routes vào group đã RequireAuth.
// VD: g.GET("/{{Domain}}", h.List)
func (h *Handler) RegisterRoutes(g *gin.RouterGroup) {
	_ = apperr.NotFound // giữ import (dùng ở handler thật)
	_ = response.OK
	_ = validation.ValidateStruct
}
`
