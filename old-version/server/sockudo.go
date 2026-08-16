package main

// Sockudo publisher — gửi events tới Sockudo (Pusher-compatible HTTP API).
// POST /apps/{app_id}/events với HMAC-SHA256 signature.

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type sockudoPublisher struct {
	url    string
	appID  string
	key    string
	secret string
	client *http.Client
}

func newSockudoPublisher(cfg config) *sockudoPublisher {
	return &sockudoPublisher{
		url:    strings.TrimRight(cfg.SockudoURL, "/"),
		appID:  cfg.SockudoAppID,
		key:    cfg.SockudoAppKey,
		secret: cfg.SockudoAppSecret,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// publish gửi một event tới channel. data được JSON-encode rồi bọc trong body
// Pusher: {"name":event,"channels":[channel],"data":"<json-string>"}.
func (p *sockudoPublisher) publish(channel, event string, data interface{}) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]interface{}{
		"name":     event,
		"channels": []string{channel},
		"data":     string(dataJSON),
	})
	if err != nil {
		return err
	}

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	md5sum := md5.Sum(body)
	bodyMD5 := hex.EncodeToString(md5sum[:])

	params := url.Values{}
	params.Set("auth_key", p.key)
	params.Set("auth_timestamp", ts)
	params.Set("auth_version", "1.0")
	params.Set("body_md5", bodyMD5)
	sorted := params.Encode() // url.Values.Encode tự sort theo key

	// stringToSign: "POST\n/apps/{app_id}/events\n<query-without-signature>"
	stringToSign := "POST\n/apps/" + p.appID + "/events\n" + sorted
	mac := hmac.New(sha256.New, []byte(p.secret))
	mac.Write([]byte(stringToSign))
	params.Set("auth_signature", hex.EncodeToString(mac.Sum(nil)))

	endpoint := fmt.Sprintf("%s/apps/%s/events?%s", p.url, p.appID, params.Encode())
	resp, err := p.client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sockudo publish %s -> %s: %w", channel, event, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("sockudo publish %s -> %s: status %d", channel, event, resp.StatusCode)
	}
	return nil
}
