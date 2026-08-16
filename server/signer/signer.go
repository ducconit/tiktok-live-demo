// Package signer generates TikTok Webcast signatures locally (no third-party
// sign server). It runs TikTok's own webmssdk.js + secsdk inside an embedded
// QuickJS engine (via cgo) and uses XHR interception to produce X-Bogus +
// X-Gnarly, then mints msToken via a request to the mssdk service.
//
// QuickJS is used instead of a pure-Go JS engine because the SDK's X-Bogus
// checksum depends on deterministic JS object-key iteration order, which
// goja (backed by Go maps) does not guarantee.
package signer

// #cgo CFLAGS: -w -D_GNU_SOURCE
// #cgo CXXFLAGS: -w -std=c++17
// #cgo LDFLAGS: -lm -lpthread -ldl
// #include <stdlib.h>
// #include "signer_c.h"
import "C"

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"unsafe"
)

//go:embed js/*.js
var jsFiles embed.FS

var jsFileNames = []string{
	"hybrid-fake-dom.js",
	"dw-index.js",
	"browser.sg.js",
	"webmssdk.js",
	"webmssdk_ex.js",
	"secsdk-lastest.umd.js",
}

// Signer holds a single QuickJS runtime with the TikTok SDK loaded. It is
// safe for concurrent use via an internal mutex.
type Signer struct {
	mu    sync.Mutex
	ptr   *C.tiktok_signer
	jsDir string
}

// New creates a Signer, extracting the embedded SDK JS files to a temp dir
// and initializing the QuickJS engine.
func New() (*Signer, error) {
	jsDir, err := extractJS()
	if err != nil {
		return nil, err
	}
	cDir := C.CString(jsDir)
	defer C.free(unsafe.Pointer(cDir))
	ptr := C.tiktok_signer_new(cDir)
	if ptr == nil {
		os.RemoveAll(jsDir)
		return nil, fmt.Errorf("signer: failed to initialize QuickJS signer")
	}
	return &Signer{ptr: ptr, jsDir: jsDir}, nil
}

// Close releases the QuickJS runtime and removes the temp JS directory.
func (s *Signer) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ptr != nil {
		C.tiktok_signer_free(s.ptr)
		s.ptr = nil
	}
	if s.jsDir != "" {
		os.RemoveAll(s.jsDir)
		s.jsDir = ""
	}
}

// SetCookies sets document.cookie inside the SDK (ttwid + msToken so the SDK
// can sign with a valid msToken).
func (s *Signer) SetCookies(cookie string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := C.CString(cookie)
	defer C.free(unsafe.Pointer(c))
	C.tiktok_signer_set_cookies(s.ptr, c)
}

// SetUserAgent overrides the fake DOM's navigator.userAgent so the
// signature's browser details match the browser_version param and the
// request's User-Agent.
func (s *Signer) SetUserAgent(ua string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := C.CString(ua)
	defer C.free(unsafe.Pointer(c))
	C.tiktok_signer_set_user_agent(s.ptr, c)
}

// Sign returns the signed URL (with X-Bogus + X-Gnarly + msToken appended).
func (s *Signer) Sign(rawURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cURL := C.CString(rawURL)
	defer C.free(unsafe.Pointer(cURL))
	cs := C.tiktok_signer_sign(s.ptr, cURL)
	if cs == nil {
		return "", fmt.Errorf("signer: signing failed: %s", C.GoString(C.tiktok_signer_last_error(s.ptr)))
	}
	defer C.tiktok_signer_free_string(cs)
	return C.GoString(cs), nil
}

func extractJS() (string, error) {
	// Use a fixed dir so a killed process (no Close) leaves at most one dir,
	// which is overwritten on the next start instead of accumulating.
	dir := filepath.Join(os.TempDir(), "tiktok-signer-js")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for _, name := range jsFileNames {
		b, err := jsFiles.ReadFile("js/" + name)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			return "", err
		}
	}
	return dir, nil
}
