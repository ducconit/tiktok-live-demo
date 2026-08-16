# ============================================================
# Makefile tiktok-live-platform — tương thích Linux + Windows (cmd.exe)
#   - Đầu file load .env (nếu có) → FORWARD_*/ADMIN_* sẵn sàng
#   - Biến Make (?=) thay shell expansion ${VAR:-def} (cmd không hỗ trợ)
#   - Lệnh OS-specific qua MKDIR (ifeq Windows_NT)
# ============================================================

# ---- Load .env nếu có (compose ports, admin defaults...) ----
# Lưu ý: .env KHÔNG được chứa $ trong VALUE (Make interpret như variable).
# Command line vẫn thắng: make admin ADMIN_EMAIL=x@y.z
ifneq (,$(wildcard .env))
include .env
endif

# ---- Defaults (không ghi đè giá trị từ .env / command line) ----
FORWARD_API_PORT           ?= 3330
FORWARD_DASHBOARD_PORT     ?= 5176
FORWARD_FRONTEND_PORT      ?= 5175
FORWARD_MAILPIT_PORT       ?= 8026
FORWARD_MINIO_CONSOLE_PORT ?= 9011
FORWARD_SOCKUDO_PORT       ?= 6002
ADMIN_EMAIL                ?= admin@example.com
ADMIN_PASSWORD             ?= admin123
ADMIN_NAME                 ?= Admin
ADMIN_ROLES                ?= system_admin
GOLANGCI_LINT              ?= golangci-lint

# mkdir portable: Windows cmd "if not exist X mkdir X", POSIX sh "mkdir -p X"
ifeq ($(OS),Windows_NT)
MKDIR := if not exist
else
MKDIR := mkdir -p
endif

.PHONY: dev down up infra logs ps migrate migrate-up migrate-down migrate-status seed sqlc-gen mock-gen test test-coverage bench admin i18n-merge lint build new-project setup watch watch-backend watch-frontend watch-dashboard

# ---- Setup (cài deps lần đầu — không cần docker) ----
setup:
	@echo "▶ backend: go mod download"
	cd backend && go mod download
	@echo "▶ frontend: bun install"
	cd frontend && bun install
	@echo "▶ dashboard: bun install"
	cd dashboard && bun install
	@echo "✅ Setup xong — chạy 'make dev' (hot reload) hoặc 'make up' (docker)"

# ---- Docker compose (dev stack) ----
# `make infra` = chỉ chạy hạ tầng (postgres/valkey/sockudo/minio/mailpit) —
# dùng kèm `make dev` (app hot reload LOCAL, kết nối tới các service này qua port host).
# `make up` = chạy TẤT CẢ trong docker (app cũng nằm trong container).
infra:
	docker compose up -d postgres valkey sockudo minio mailpit
	@echo "▶ Infra sẵn sàng: postgres :5433 · valkey :6380 · sockudo :6002 · minio :9010/9011 · mailpit :8026"
	@echo "▶ Chạy 'make dev' để start app local (air + bun dev)"

up:
	docker compose up -d --build
	@echo "Backend : http://localhost:$(FORWARD_API_PORT)"
	@echo "Dashboard: http://localhost:$(FORWARD_DASHBOARD_PORT)"
	@echo "Frontend : http://localhost:$(FORWARD_FRONTEND_PORT)"
	@echo "Sockudo  : http://localhost:$(FORWARD_SOCKUDO_PORT)"
	@echo "Mailpit : http://localhost:$(FORWARD_MAILPIT_PORT)"
	@echo "MinIO   : http://localhost:$(FORWARD_MINIO_CONSOLE_PORT)"

# ---- Dev local (hot reload) — `make dev` = chạy song song 3 tiến trình: ----
# backend (air) + frontend + dashboard (bun dev). Ctrl-C dừng tất cả.
# Port local: frontend $(FORWARD_FRONTEND_PORT) / dashboard $(FORWARD_DASHBOARD_PORT)
# — khác port mặc định 5173 của vite để chạy được cả 2 app cùng lúc.
dev:
	@echo "▶ Backend  : cd backend && air (hot reload) — cần infra (make infra)"
	@echo "▶ Frontend : http://localhost:$(FORWARD_FRONTEND_PORT) (bun dev)"
	@echo "▶ Dashboard: http://localhost:$(FORWARD_DASHBOARD_PORT) (bun dev)"
	@echo "▶ Ctrl-C để dừng tất cả"
	@trap 'kill 0' INT TERM EXIT; \
	(cd backend && CGO_ENABLED=1 air) & \
	(cd frontend && bun run dev --port $(FORWARD_FRONTEND_PORT)) & \
	(cd dashboard && bun run dev --port $(FORWARD_DASHBOARD_PORT)) & \
	wait

# Alias: make watch = make dev (chạy hot reload song song)
watch: dev

watch-backend:
	@echo "▶ Backend — air (hot reload) — CGO_ENABLED=1 (QuickJS signer)"
	cd backend && CGO_ENABLED=1 air

watch-frontend:
	@echo "▶ Frontend — http://localhost:$(FORWARD_FRONTEND_PORT)"
	cd frontend && bun run dev --port $(FORWARD_FRONTEND_PORT)

watch-dashboard:
	@echo "▶ Dashboard — http://localhost:$(FORWARD_DASHBOARD_PORT)"
	cd dashboard && bun run dev --port $(FORWARD_DASHBOARD_PORT)

down:
	docker compose down

logs:
	docker compose logs -f --tail=100

ps:
	docker compose ps

# ---- Migration (chạy trong container backend) ----
# Lệnh migrate nằm trong binary app (cmd/app/commands/migrate.go) — dùng chung
# cho dev (go run) lẫn production (./gvs migrate up sau khi build).
# migrate = alias của migrate up (dùng quen: make migrate)
migrate: migrate-up

migrate-up:
	docker compose exec backend go run ./cmd/app migrate up

migrate-down:
	docker compose exec backend go run ./cmd/app migrate down

migrate-status:
	docker compose exec backend go run ./cmd/app migrate status

seed:
	docker compose exec backend go run ./cmd/devtool seed

sqlc-gen:
	cd backend && sqlc generate

mock-gen:
	cd backend && mockery

# ---- Test / lint / coverage / benchmark ----
test:
	cd backend && go test ./... -race

test-coverage:
	cd backend && go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1
	@echo "→ Report HTML: cd backend && go tool cover -html=coverage.out"

bench:
	cd backend && go test -bench=. -benchmem -run=^$$ ./core/... ./internal/...

# Tạo admin (make:admin) — mặc định admin@example.com / admin123, role system_admin.
# Override: make admin ADMIN_EMAIL=... ADMIN_PASSWORD=... ADMIN_NAME=... ADMIN_ROLES=admin,editor
admin:
	cd backend && go run ./cmd/devtool make:admin \
		--name "$(ADMIN_NAME)" \
		--email "$(ADMIN_EMAIL)" \
		--password "$(ADMIN_PASSWORD)" \
		--roles "$(ADMIN_ROLES)"

# i18n: tạo/merge message files — sau khi THÊM message ID vào code:
#   vi.json (nguồn) + en.json → translate.*.json = danh sách key cần dịch
#   (workflow đầy đủ: docs/i18n.md)
i18n-merge:
	cd backend && $(MKDIR) .i18n && goi18n merge -sourceLanguage vi -outdir .i18n -format json core/i18n/messages/vi.json core/i18n/messages/en.json && echo "→ .i18n/translate.*.json = key cần dịch; merge xong đưa vào messages/"

lint:
	cd backend && $(GOLANGCI_LINT) run ./...

# ---- Build (version/branch-aware) ----
# Version: VERSION env override → branch release-* → tag gần nhất → 1.0.0
# Fallback portable (Windows không có scripts/version.sh / GNU date)
BUILD_VERSION ?= $(shell ./scripts/version.sh 2>/dev/null || echo 1.0.0)
BUILD_HASH    := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
BUILD_DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null)

build:
	cd backend && go build -tags sonic -ldflags "-X main.version=$(BUILD_VERSION) -X main.buildHash=$(BUILD_HASH) -X main.buildDate=$(BUILD_DATE)" -o bin/gvs ./cmd/app
	@echo "✅ Built backend/bin/gvs (version=$(BUILD_VERSION) hash=$(BUILD_HASH) date=$(BUILD_DATE))"

# ---- Tạo project mới từ template ----
new-project:
	cd backend && go run ./cmd/devtool new:project $(NAME) "$(TITLE)"
