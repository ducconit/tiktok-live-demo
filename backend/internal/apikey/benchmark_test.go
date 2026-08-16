package apikey

import (
	"testing"
)

// BenchmarkHashKey — sha256(prefix.key) — hot path của middleware verify
// (mỗi request API key đều hash + lookup).
func BenchmarkHashKey(b *testing.B) {
	key := "gvs_live_ab12cd34ef56gh78ij90kl12mn34op56qr78st90uv"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hashKey(key)
	}
}

func BenchmarkDisplayPrefix(b *testing.B) {
	key := "gvs_live_ab12cd34ef56gh78ij90kl12mn34op56qr78st90uv"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = displayPrefix(key)
	}
}
