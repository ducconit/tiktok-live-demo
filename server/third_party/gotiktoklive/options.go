package gotiktoklive

import "net/http"

type TikTokLiveOption func(t *TikTok) error

// SignFunc signs a webcast URL and fetches it, returning the response body and
// headers. It mirrors the Euler signer's /webcast/fetch/ behavior: the caller
// passes an unsigned URL and gets back the signed fetch response (protobuf
// body + headers incl. X-Set-TT-Cookie). A nil return body + nil error means
// the signer produced no data (e.g. soft-blocked).
type SignFunc func(reqUrl string) ([]byte, http.Header, error)

// SigningFunc installs a self-hosted signer, replacing the default Euler
// Stream signer. Disables signer rate-limit validation (not applicable to a
// local signer).
func SigningFunc(f SignFunc) TikTokLiveOption {
	return func(t *TikTok) error {
		t.signFunc = f
		t.getLimits = false
		return nil
	}
}

// SignURLFunc signs a URL (returning the signed URL) without fetching it.
// Used for the WebSocket handshake, which must be signed but is dialed with a
// WebSocket client rather than fetched over HTTP.
type SignURLFunc func(reqUrl string) (string, error)

// SigningURLFunc installs a sign-only function for the WebSocket URL.
func SigningURLFunc(f SignURLFunc) TikTokLiveOption {
	return func(t *TikTok) error {
		t.signURLFunc = f
		return nil
	}
}

// EnableExperimentalEvents enables experimental events that have not been figured out yet and the API for them is not
// stable. It may also induce additional logging that might be undesirable.
func EnableExperimentalEvents(t *TikTok) error {
	t.enableExperimentalEvents = true
	return nil
}

// EnableExtraWebCastDebug an unreasonable amount of debug for library development and troubleshooting. This option
// makes no guarantee of ever having the same output and is only for development and triage purposes.
func EnableExtraWebCastDebug(t *TikTok) error {
	t.enableExtraDebug = true
	t.c.Transport = &loggingTransport{Transport: http.DefaultTransport}
	return nil
}

// EnableWSTrace will put traces for all websocket messages into the given file. The file will be overwritten so
// if you want multiple traces make sure handle giving a unique filename each startup.
func EnableWSTrace(file string) TikTokLiveOption {
	return func(t *TikTok) error {
		t.enableWSTrace = true
		t.wsTraceFile = file
		t.wsTraceChan = make(chan struct{ direction, hex string }, 50)
		return nil
	}
}

// Proxy will set a proxy for both the http client and the websocket. You can
// manually set a proxy with option or by using the HTTPS_PROXY environment variable.
// ALL_PROXY can be used to set a proxy only for the websocket.
func Proxy(url string, insecure bool) TikTokLiveOption {
	if url == "" {
		return func(t *TikTok) error {
			return nil
		}
	}
	return func(t *TikTok) error {
		return t.setProxy(url, insecure)
	}
}
