#!/usr/bin/env bash
# ============================================================
# version.sh — sinh version cho build (chạy bởi `make build`).
#
# Ưu tiên:
#   1. $VERSION env (chỉ định tay) — vd: VERSION=2.5 make build
#   2. Branch release-* (production) — phần sau prefix: release-1.0 → "1.0"
#   3. Branch khác (main = staging) — tag gần nhất (git describe)
#   4. Không có tag (build lần đầu) — "1.0.0"
# ============================================================
set -euo pipefail

if [[ -n "${VERSION:-}" ]]; then
  echo "$VERSION"
  exit 0
fi

BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")"

if [[ "$BRANCH" == release-* ]]; then
  echo "${BRANCH#release-}"
  exit 0
fi

TAG="$(git describe --tags --abbrev=0 2>/dev/null || true)"
if [[ -n "$TAG" ]]; then
  echo "${TAG#v}"   # bỏ prefix "v" nếu tag dạng v1.2.0
  exit 0
fi

echo "1.0.0"
