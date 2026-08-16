package signer

import (
	"strings"
	"testing"
)

const testUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

const testURL = "https://webcast.tiktok.com/webcast/im/fetch/?aid=1988&room_id=1234567890123456789&resp_content_type=protobuf&identity=audience&live_id=12&sup_ws_ds_opt=1&cursor=&internal_ext="

// TestSignerProducesSignatures verifies the QuickJS signer appends X-Bogus and
// X-Gnarly to the URL.
func TestSignerProducesSignatures(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()
	s.SetUserAgent(testUA)
	s.SetCookies("ttwid=test; msToken=test")

	signed, err := s.Sign(testURL)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !strings.Contains(signed, "X-Bogus=") {
		t.Fatalf("missing X-Bogus: %s", signed)
	}
	if !strings.Contains(signed, "X-Gnarly=") {
		t.Fatalf("missing X-Gnarly: %s", signed)
	}
}

// TestSignerXbogusDeterministic verifies X-Bogus is stable for the same input
// (this was broken with goja — the whole reason we switched to QuickJS).
func TestSignerXbogusDeterministic(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()
	s.SetUserAgent(testUA)
	s.SetCookies("ttwid=test; msToken=test")

	first, err := s.Sign(testURL)
	if err != nil {
		t.Fatalf("Sign 1: %v", err)
	}
	second, err := s.Sign(testURL)
	if err != nil {
		t.Fatalf("Sign 2: %v", err)
	}
	if xb := extractParam(first, "X-Bogus"); xb != extractParam(second, "X-Bogus") {
		t.Fatalf("X-Bogus not deterministic: %s vs %s", xb, extractParam(second, "X-Bogus"))
	}
}

func extractParam(u, key string) string {
	for _, p := range strings.Split(u, "&") {
		if strings.HasPrefix(p, key+"=") {
			return strings.TrimPrefix(p, key+"=")
		}
	}
	return ""
}
