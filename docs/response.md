# Chuẩn response — envelope 4 key

> Khớp với `core/response` + `core/apperr` — cập nhật khi code đổi.

## Shape

**MỌI response** (thành công lẫn lỗi) là 1 JSON envelope 4 key:

```json
{
  "code": "0",
  "msg": "",
  "data": null,
  "meta": {}
}
```

| Key | Ý nghĩa |
|---|---|
| `code` | **Luôn là STRING**: `"0"` = thành công; lỗi = HTTP status dạng string (`"400"`, `"401"`, ..., `"500"`, `"503"`) |
| `msg` | Thông báo (rỗng khi thành công; lỗi internal luôn msg mặc định, không lộ chi tiết) |
| `data` | Dữ liệu chính — `null` khi không có; **danh sách luôn là mảng** (`[]` khi rỗng) |
| `meta` | Phân trang `{limit, page, total}` **hoặc** validation `{field: "message"}` |

## Bảng code

| code | HTTP status | Ý nghĩa |
|---|---|---|
| `"0"` | 200/201/204 | thành công (msg bỏ trống) |
| `"400"` | 400 | bad request (JSON sai, fallback validation không rõ field) |
| `"401"` | 401 | thiếu/sai token, API key hết hạn/revoked |
| `"403"` | 403 | không có quyền (permission/scope) |
| `"404"` | 404 | không tìm thấy |
| `"413"` | 413 | file quá lớn |
| `"422"` | 422 | validation — `meta = {field: "message lỗi"}` (tên field theo JSON tag) |
| `"429"` | 429 | rate limit |
| `"500"` | 500 | internal — msg mặc định `"Ops! Đang có lỗi xảy ra, vui lòng thử lại sau."` (KHÔNG lộ chi tiết) |
| `"503"` | 503 | maintenance mode — chặn mọi API trừ `/config` |

## Ví dụ

Thành công (phân trang):
```json
{ "code": "0", "msg": "", "data": [ ... ], "meta": { "limit": 10, "page": 1, "total": 42 } }
```

Validation (422):
```json
{ "code": "422", "msg": "Dữ liệu không hợp lệ", "data": null,
  "meta": { "email": "Email không hợp lệ", "password": "Mật khẩu tối thiểu 8 ký tự" } }
```

Internal (500 — không lộ chi tiết):
```json
{ "code": "500", "msg": "Ops! Đang có lỗi xảy ra, vui lòng thử lại sau.", "data": null, "meta": {} }
```

## Dùng trong code

```go
// Thành công
response.OK(c, data)              // 200
response.OKList(c, items)         // 200 — data luôn là mảng ([] nếu nil)
response.OKWithMeta(c, items, meta) // 200 — phân trang
response.Created(c, data)         // 201
response.NoContent(c)             // 204

// Lỗi
response.Error(c, err)            // *AppError → tự map code/status
// Tạo lỗi:
apperr.BadRequest("invalid_json", "Request body không hợp lệ")
apperr.Unauthorized("invalid_token", "...")
apperr.Validation(map[string]string{"email": "Email không hợp lệ"}) // 422
apperr.WrapInternal(err)          // 500 — KHÔNG lộ err.Error() ra response
```

## Lưu ý

- **Tên field trong meta validation** = JSON tag của struct (snake_case), không phải tên Go.
- **Mọi lỗi phải log**: middleware tự ghi ERROR (≥500, kèm details) / WARN (4xx, kèm code) — xem `docs/logger.md`.
- Client nên đọc `code` (string) để phân nhánh, không dựa vào HTTP status duy nhất.
