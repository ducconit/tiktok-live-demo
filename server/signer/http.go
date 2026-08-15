package signer

import (
	"context"
	"net"
	"net/http"
	"time"

	utls "github.com/refraction-networking/utls"
)

// chromeSpec returns a Chrome ClientHello spec forced to HTTP/1.1 (no "h2"),
// because Go's http.Transport can't speak HTTP/2 over a non-*tls.Conn.
func chromeSpec() *utls.ClientHelloSpec {
	spec, err := utls.UTLSIdToSpec(utls.HelloChrome_Auto)
	if err != nil {
		return nil
	}
	for i, ext := range spec.Extensions {
		if alpn, ok := ext.(*utls.ALPNExtension); ok {
			alpn.AlpnProtocols = []string{"http/1.1"}
			spec.Extensions[i] = alpn
		}
	}
	return &spec
}

// ChromeClient returns an http.Client whose TLS handshake mimics Chrome, so
// TikTok's WAF (which fingerprints JA3/JA4) does not soft-block requests.
func ChromeClient() *http.Client {
	spec := chromeSpec()
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 15 * time.Second,
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 15 * time.Second}
			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			host, _, _ := net.SplitHostPort(addr)
			uconn := utls.UClient(conn, &utls.Config{ServerName: host}, utls.HelloCustom)
			if spec != nil {
				if err := uconn.ApplyPreset(spec); err != nil {
					conn.Close()
					return nil, err
				}
			}
			if err := uconn.Handshake(); err != nil {
				conn.Close()
				return nil, err
			}
			return uconn, nil
		},
	}
	return &http.Client{Transport: tr, Timeout: 30 * time.Second}
}
