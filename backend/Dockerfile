# Prod image — binary (CGO cho QuickJS signer — live module), non-root
FROM golang:1.26-alpine AS builder

# build-base (g++/make) — bắt buộc cho QuickJS signer (CGO) trong internal/live
RUN apk add --no-cache build-base ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
COPY third_party ./third_party
RUN go mod download
COPY . .

# build tag sonic = dùng bytedance/sonic làm JSON engine cho gin
# gvs = ứng dụng deploy (tên binary — cmd/app); devtool = CLI tools (migrate/seed/...) — dùng chung binary
# CGO_ENABLED=1 (QuickJS signer); binary link musl → chạy trên alpine base
RUN CGO_ENABLED=1 GOOS=linux go build -tags sonic -ldflags="-s -w" -o /out/gvs ./cmd/app \
    && CGO_ENABLED=1 GOOS=linux go build -tags sonic -ldflags="-s -w" -o /out/devtool ./cmd/devtool

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /out/gvs /out/devtool /app/

USER app
EXPOSE 3330

# ENTRYPOINT ["/app/gvs", "--port", "3300"] — override qua docker compose ở stage/prod
