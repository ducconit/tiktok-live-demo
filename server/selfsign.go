package main

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/tiktok-bar/server/signer"
)

// signUA must match the QuickJS signer's fake-DOM navigator.userAgent
// (Chrome 131 Mac) so the X-Gnarly md5(userAgent) is consistent with the
// fetch request's User-Agent header.
const signUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// selfSigner implements gotiktoklive.SignFunc using the local QuickJS signer
// (X-Bogus + X-Gnarly + msToken) plus a Chrome-fingerprint TLS client. It
// mirrors the Euler signer's behavior: sign the URL, fetch it, return the
// response body + headers.
type selfSigner struct {
	mu      sync.Mutex
	signer  *signer.Signer
	client  tls_client.HttpClient
	ttwid   string
	msToken string
}

func newSelfSigner() (*selfSigner, error) {
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithClientProfile(profiles.Chrome_131),
		tls_client.WithTimeout(30),
	)
	if err != nil {
		return nil, err
	}

	s, err := signer.New()
	if err != nil {
		return nil, err
	}
	s.SetUserAgent(signUA)

	ss := &selfSigner{signer: s, client: client}

	// Warmup: obtain a fresh ttwid cookie (TikTok soft-blocks without one).
	// net/http's cookie jar reliably captures Set-Cookie (the fhttp client
	// does not), so warmup via net/http and reuse the ttwid for the fetch.
	ss.warmup()

	// Mint an initial msToken; it rotates on every im/fetch response.
	if tok, err := signer.MintMsToken(); err == nil {
		ss.msToken = tok
	}

	return ss, nil
}

func (ss *selfSigner) warmup() {
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar, Timeout: 15e9}
	for _, u := range []string{"https://www.tiktok.com/", "https://www.tiktok.com/foryou"} {
		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Set("User-Agent", signUA)
		if resp, err := c.Do(req); err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}
	u, _ := url.Parse("https://www.tiktok.com")
	for _, ck := range jar.Cookies(u) {
		if ck.Name == "ttwid" {
			ss.ttwid = ck.Value
			return
		}
	}
}

func (ss *selfSigner) close() {
	if ss != nil && ss.signer != nil {
		ss.signer.Close()
	}
}

// signOnly signs reqUrl and returns the signed URL (no fetch). Used for the
// WebSocket handshake URL.
func (ss *selfSigner) signOnly(reqUrl string) (string, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.signer.SetCookies("ttwid=" + ss.ttwid + "; msToken=" + ss.msToken)
	return ss.signer.Sign(reqUrl)
}

// signFetch signs reqUrl and fetches it, returning the response body and a
// net/http.Header carrying the X-Set-TT-Cookie (and other) headers.
func (ss *selfSigner) signFetch(reqUrl string) ([]byte, http.Header, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.signer.SetCookies("ttwid=" + ss.ttwid + "; msToken=" + ss.msToken)

	signed, err := ss.signer.Sign(reqUrl)
	if err != nil {
		return nil, nil, err
	}

	req, err := fhttp.NewRequest("GET", signed, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", signUA)
	req.Header.Set("Referer", "https://www.tiktok.com/")
	req.Header.Set("Origin", "https://www.tiktok.com")
	req.Header.Set("Accept", "text/html,application/json,application/protobuf")
	req.Header.Set("Cookie", "ttwid="+ss.ttwid+"; msToken="+ss.msToken)

	resp, err := ss.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	// Rotate msToken from the response so the next fetch signs fresh.
	if mt := resp.Header.Get("X-Ms-Token"); mt != "" {
		ss.msToken = mt
	}

	// Convert fhttp.Header -> net/http.Header (SignFunc contract).
	headers := make(http.Header)
	for k, vs := range resp.Header {
		for _, v := range vs {
			headers.Add(k, v)
		}
	}

	return body, headers, nil
}
