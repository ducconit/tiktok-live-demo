package gotiktoklive

import "net/http"

type TikTokLiveOption func(t *TikTok) error

// SignFunc signs a webcast URL and fetches it, returning the response body and
// headers (protobuf body + headers incl. X-Set-TT-Cookie). A nil return body +
// nil error means the signer produced no data (e.g. soft-blocked).
type SignFunc func(reqUrl string) ([]byte, http.Header, error)

// SigningFunc installs a self-hosted signer.
func SigningFunc(f SignFunc) TikTokLiveOption {
	return func(t *TikTok) error {
		t.signFunc = f
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
