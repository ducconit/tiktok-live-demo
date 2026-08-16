// Package migrations — embed toàn bộ file SQL migration để goose dùng chung
// cho cả API server lẫn CLI (binary đơn, không cần mount thư mục).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
