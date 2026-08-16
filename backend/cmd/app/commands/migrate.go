package commands

import (
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/spf13/cobra"
)

// MigrateCmd — quản lý migration database cho binary app (production):
//
//	./app migrate up      # chạy toàn bộ migration chưa áp dụng
//	./app migrate down    # lùi 1 bước
//	./app migrate status  # xem trạng thái
//
// Migration engine: goose v3 (xem docs/migration.md). Config đọc từ .env
// (cạnh binary / CWD) — giống lệnh khác; migration files embed trong binary
// qua migrations.FS nên không cần source tree khi chạy.
func MigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Quản lý migration database (up/down/status)",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "up",
			Short: "Chạy toàn bộ migration chưa áp dụng",
			RunE: func(cmd *cobra.Command, _ []string) error {
				cfg, err := loadCfg()
				if err != nil {
					return err
				}
				return database.Migrate(cmd.Context(), cfg.Cfg.Database.URL)
			},
		},
		&cobra.Command{
			Use:   "down",
			Short: "Lùi 1 bước migration",
			RunE: func(cmd *cobra.Command, _ []string) error {
				cfg, err := loadCfg()
				if err != nil {
					return err
				}
				return database.MigrateDown(cmd.Context(), cfg.Cfg.Database.URL)
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Xem trạng thái migration",
			RunE: func(cmd *cobra.Command, _ []string) error {
				cfg, err := loadCfg()
				if err != nil {
					return err
				}
				return database.MigrateStatus(cmd.Context(), cfg.Cfg.Database.URL)
			},
		},
	)
	return cmd
}
