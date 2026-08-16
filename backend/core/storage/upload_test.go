package storage

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeFileHeader — tạo multipart.FileHeader giả (không cần HTTP request).
func makeFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", "image/png")
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	// Parse lại để lấy FileHeader với Open() hoạt động
	reader := multipart.NewReader(&buf, w.Boundary())
	form, err := reader.ReadForm(1 << 20)
	require.NoError(t, err)
	fhs := form.File["file"]
	require.Len(t, fhs, 1)
	return fhs[0]
}

func TestUploadImage_Success(t *testing.T) {
	d := newTestLocal(t, "/storage")
	fh := makeFileHeader(t, "avatar.png", bytes.Repeat([]byte{0x89, 0x50, 0x4e, 0x47}, 10))

	key, url, err := UploadImage(context.Background(), d, "users/abc/avatars", fh)
	require.NoError(t, err)
	// Key = folder + hash sha256(16 hex) + ext
	assert.Contains(t, key, "users/abc/avatars/")
	assert.Contains(t, key, ".png")
	assert.NotContains(t, key, "uuid")
	assert.Len(t, key, len("users/abc/avatars/")+16+len(".png"))
	assert.Equal(t, "/storage/"+key, url)
	assert.True(t, d.Exists(key))

	// Upload lại cùng nội dung → CÙNG key (dedup bằng hash)
	key2, _, err := UploadImage(context.Background(), d, "users/abc/avatars", fh)
	require.NoError(t, err)
	assert.Equal(t, key, key2)

	got, err := d.Get(key)
	require.NoError(t, err)
	assert.Equal(t, int64(40), int64(len(got)))
}

func TestUploadImage_TooLarge(t *testing.T) {
	d := newTestLocal(t, "/storage")
	fh := makeFileHeader(t, "big.png", bytes.Repeat([]byte{0x01}, maxAvatarSize+1))
	_, _, err := UploadImage(context.Background(), d, "users/abc/avatars", fh)
	assert.True(t, errors.Is(err, ErrFileTooLarge))
}

func TestUploadImage_UnsupportedExt(t *testing.T) {
	d := newTestLocal(t, "/storage")
	fh := makeFileHeader(t, "evil.svg", []byte("<script>alert(1)</script>"))
	_, _, err := UploadImage(context.Background(), d, "users/abc/avatars", fh)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "không hỗ trợ")
	assert.False(t, d.Exists("users/abc/avatars/"))
}
