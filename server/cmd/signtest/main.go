package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	fhttp "github.com/bogdanfinn/fhttp"
	gotiktoklive "github.com/steampoweredtaco/gotiktoklive"
	"github.com/tiktok-bar/server/signer"
)

const ua = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

func main() {
	username := "icegirls_01"
	if len(os.Args) > 1 {
		username = os.Args[1]
	}

	s, _ := signer.New()
	defer s.Close()
	s.SetUserAgent(ua)

	client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
		tls_client.WithClientProfile(profiles.Chrome_131),
		tls_client.WithTimeout(30),
	)

	// warmup via net/http (reliable ttwid capture)
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Jar: jar}
	for _, u := range []string{"https://www.tiktok.com/", "https://www.tiktok.com/foryou"} {
		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Set("User-Agent", ua)
		if resp, err := hc.Do(req); err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}
	u, _ := url.Parse("https://www.tiktok.com")
	ttwid := ""
	for _, ck := range jar.Cookies(u) {
		if ck.Name == "ttwid" {
			ttwid = ck.Value
		}
	}
	msToken, _ := signer.MintMsToken()

	signFetch := func(reqUrl string) ([]byte, http.Header, error) {
		s.SetCookies("ttwid=" + ttwid + "; msToken=" + msToken)
		signed, err := s.Sign(reqUrl)
		if err != nil {
			return nil, nil, err
		}
		req, _ := fhttp.NewRequest("GET", signed, nil)
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Referer", "https://www.tiktok.com/")
		req.Header.Set("Origin", "https://www.tiktok.com")
		req.Header.Set("Accept", "text/html,application/json,application/protobuf")
		req.Header.Set("Cookie", "ttwid="+ttwid+"; msToken="+msToken)
		resp, err := client.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if mt := resp.Header.Get("X-Ms-Token"); mt != "" {
			msToken = mt
		}
		h := make(http.Header)
		for k, vs := range resp.Header {
			for _, v := range vs {
				h.Add(k, v)
			}
		}
		return body, h, nil
	}
	signOnly := func(reqUrl string) (string, error) {
		s.SetCookies("ttwid=" + ttwid + "; msToken=" + msToken)
		return s.Sign(reqUrl)
	}

	t, err := gotiktoklive.NewTikTok(
		gotiktoklive.DisableSigningLimitsValidation,
		gotiktoklive.SigningFunc(signFetch),
		gotiktoklive.SigningURLFunc(signOnly),
	)
	if err != nil {
		fmt.Println("NewTikTok err:", err)
		return
	}
	t.SetErrorHandler(func(v ...interface{}) { fmt.Println("[err]", v) })
	t.SetInfoHandler(func(v ...interface{}) { fmt.Println("[info]", v) })

	fmt.Println("tracking", username, "...")
	live, err := t.TrackUser(username)
	if err != nil {
		fmt.Println("TrackUser err:", err)
		return
	}
	defer live.Close()
	fmt.Println("connected, roomId:", live.ID)

	timeout := time.After(30 * time.Second)
	count := 0
	for {
		select {
		case ev := <-live.Events:
			count++
			fmt.Printf("event #%d: %T\n", count, ev)
			if count >= 5 {
				fmt.Println("got 5 events, done")
				return
			}
		case <-timeout:
			fmt.Printf("timeout, got %d events\n", count)
			return
		}
	}
}
