package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLocal(t *testing.T, url string) *LocalDisk {
	t.Helper()
	d, err := NewLocalDisk("test", filepath.Join(t.TempDir(), "root"), url)
	require.NoError(t, err)
	return d
}

func TestLocalDisk_PutGetDelete(t *testing.T) {
	d := newTestLocal(t, "/storage")

	require.NoError(t, d.Put("dir/file.txt", []byte("hello")))
	got, err := d.Get("dir/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))

	assert.True(t, d.Exists("dir/file.txt"))
	size, err := d.Size("dir/file.txt")
	require.NoError(t, err)
	assert.Equal(t, int64(5), size)

	require.NoError(t, d.Delete("dir/file.txt"))
	assert.False(t, d.Exists("dir/file.txt"))
	_, err = d.Get("dir/file.txt")
	require.Error(t, err) // đã xoá
}

func TestLocalDisk_Overwrite(t *testing.T) {
	d := newTestLocal(t, "/storage")
	require.NoError(t, d.Put("a.txt", []byte("1")))
	require.NoError(t, d.Put("a.txt", []byte("22")))
	got, _ := d.Get("a.txt")
	assert.Equal(t, "22", string(got))
}

func TestLocalDisk_URL(t *testing.T) {
	// public: url prefix + name
	pub := newTestLocal(t, "/storage")
	assert.Equal(t, "/storage/avatars/a.png", pub.URL("avatars/a.png"))
	assert.Equal(t, "/storage/x", pub.URL("x"))

	// private: url rỗng → URL ""
	priv := newTestLocal(t, "")
	assert.Equal(t, "", priv.URL("secret.txt"))
}

func TestLocalDisk_TemporaryURL_EqualsURL(t *testing.T) {
	d := newTestLocal(t, "/storage")
	u, err := d.TemporaryURL("a.txt", 0)
	require.NoError(t, err)
	assert.Equal(t, "/storage/a.txt", u) // local không presign
}

func TestLocalDisk_PathTraversal(t *testing.T) {
	d := newTestLocal(t, "/storage")
	for _, evil := range []string{"../evil.txt", "a/../../evil.txt", "/etc/passwd", "..\\evil.txt", "."} {
		err := d.Put(evil, []byte("x"))
		assert.True(t, errors.Is(err, ErrInvalidPath), "phải chặn: %q (got %v)", evil, err)
		assert.False(t, d.Exists(evil), "không được tồn tại: %q", evil)
	}
	// File ngoài root không bị đọc qua Get
	_, err := d.Get("../../etc/passwd")
	assert.True(t, errors.Is(err, ErrInvalidPath))
}

func TestLocalDisk_Delete_NotExistNoError(t *testing.T) {
	d := newTestLocal(t, "")
	require.NoError(t, d.Delete("không-tồn-tại.txt"))
}

func TestLocalDisk_Health(t *testing.T) {
	d := newTestLocal(t, "")
	require.NoError(t, d.Health(t.Context()))
	// Health khi root bị xoá → lỗi
	require.NoError(t, os.RemoveAll(d.Root()))
	require.Error(t, d.Health(t.Context()))
}
