package gotiktoklive

// WebSocket transport for the webcast push server, modelled on key2/ttlive-cpp:
//   - build the ws URL from base_ws_params (NOT the im/fetch PushServer), sign it
//   - send an im_enter_room frame after the handshake
//   - heartbeat every 10s
//   - parse WebcastPushFrame -> (gzip?) WebcastResponse -> events, ack when needed

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	pb "github.com/steampoweredtaco/gotiktoklive/proto"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"golang.org/x/net/proxy"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

const wsHeartbeatInterval = 10 * time.Second

// baseWsParams are the WebSocket query params TikTok's web player sends.
func baseWsParams(roomID string) url.Values {
	vs := url.Values{}
	for k, v := range map[string]string{
		"version_code":        "270000",
		"device_platform":     "web",
		"cookie_enabled":      "true",
		"app_name":            "tiktok_web",
		"aid":                 "1988",
		"live_id":             "12",
		"identity":            "audience",
		"sup_ws_ds_opt":       "1",
		"ws_direct":           "1",
		"resp_content_type":   "protobuf",
		"did_rule":            "3",
		"heartbeat_duration":  "10000",
		"last_rtt":            "0",
		"app_language":        "en",
		"webcast_language":    "en",
		"client_enter":        "1",
		"update_version_code": "2.0.0",
		"browser_name":        "Mozilla",
		"browser_online":      "true",
		"room_id":             roomID,
		"compress":            "gzip",
	} {
		vs.Set(k, v)
	}
	return vs
}

// wsHost maps the tt-target-idc cookie to a webcast-ws region host.
func (t *TikTok) wsHost() string {
	u, _ := url.Parse("https://www.tiktok.com")
	for _, c := range t.c.Jar.Cookies(u) {
		if c.Name == "tt-target-idc" {
			idc := c.Value
			if len(idc) >= 2 && idc[:2] == "eu" {
				return "webcast-ws.eu.tiktok.com"
			}
			if len(idc) >= 2 && (idc[:2] == "sg" || idc[:2] == "al") {
				return "webcast-ws.sg.tiktok.com"
			}
		}
	}
	return "webcast-ws.us.tiktok.com"
}

// buildWSUrl builds + signs the wss:// URL for the push server.
func (l *Live) buildWSUrl() (string, error) {
	vs := baseWsParams(l.ID)
	if l.cursor != "" {
		vs.Set("cursor", l.cursor)
	}
	base := fmt.Sprintf("https://%s/webcast/im/ws_proxy/ws_reuse_supplement/?%s",
		l.t.wsHost(), vs.Encode())

	if l.t.signURLFunc == nil {
		return "", fmt.Errorf("no signURLFunc configured for websocket")
	}
	signed, err := l.t.signURLFunc(base)
	if err != nil {
		return "", err
	}
	if len(signed) >= 8 && signed[:8] == "https://" {
		signed = "wss://" + signed[8:]
	}
	return signed, nil
}

// makePushFrame serialises a WebcastPushFrame manually (fields 2,6,7,8).
func makePushFrame(payloadType string, payload []byte, logID uint64) []byte {
	var b []byte
	if logID != 0 {
		b = protowire.AppendTag(b, 2, protowire.VarintType)
		b = protowire.AppendVarint(b, logID)
	}
	b = protowire.AppendTag(b, 6, protowire.BytesType)
	b = protowire.AppendString(b, "pb")
	b = protowire.AppendTag(b, 7, protowire.BytesType)
	b = protowire.AppendString(b, payloadType)
	if len(payload) > 0 {
		b = protowire.AppendTag(b, 8, protowire.BytesType)
		b = protowire.AppendBytes(b, payload)
	}
	return b
}

// encodeHeartBeat serialises HeartBeatMessage{room_id, send_packet_seq_id}.
func encodeHeartBeat(roomID int64, seq int64) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(roomID))
	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(seq))
	return b
}

// encodeImEnterRoom serialises WebcastImEnterRoomMessage{room_id, identity,
// cursor} (fields 1,5,6).
func encodeImEnterRoom(roomID int64) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(roomID))
	b = protowire.AppendTag(b, 5, protowire.BytesType)
	b = protowire.AppendString(b, "audience")
	b = protowire.AppendTag(b, 6, protowire.BytesType)
	b = protowire.AppendString(b, "")
	return b
}

func gunzipData(b []byte) ([]byte, error) {
	if len(b) < 2 || b[0] != 0x1f || b[1] != 0x8b {
		return b, nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

// connectWebSocket dials the signed wss URL, sends im_enter_room, then starts
// the read + heartbeat goroutines. Events flow into l.Events; the read
// goroutine closes l.Events when the session ends.
func (l *Live) connectWebSocket() error {
	wsURL, err := l.buildWSUrl()
	if err != nil {
		return err
	}

	u, _ := url.Parse("https://www.tiktok.com")
	cookies := l.t.c.Jar.Cookies(u)
	var cookieStr string
	for _, c := range cookies {
		if cookieStr == "" {
			cookieStr = c.String()
		} else {
			cookieStr += "; " + c.String()
		}
	}
	headers := http.Header{}
	if cookieStr != "" {
		headers.Set("Cookie", cookieStr)
	}
	headers.Set("User-Agent", userAgent)

	var proxyURI *url.URL
	for _, envVar := range []string{"HTTP_PROXY", "HTTPS_PROXY"} {
		if v := os.Getenv(envVar); v != "" {
			if uri, err := url.Parse(v); err == nil {
				proxyURI = uri
				break
			}
		}
	}
	if l.t.proxy != nil {
		proxyURI = l.t.proxy
	}
	proxyDial := proxy.FromEnvironment()
	if proxyURI != nil {
		if d, err := proxy.FromURL(proxyURI, proxy.Direct); err == nil {
			proxyDial = d
		}
	}

	dialer := ws.Dialer{
		Header:    ws.HandshakeHeaderHTTP(headers),
		NetDial:   func(ctx context.Context, network, addr string) (net.Conn, error) { return proxyDial.Dial(network, addr) },
		Protocols: []string{"echo-protocol"},
		Timeout:   15 * time.Second,
	}
	conn, _, _, err := dialer.Dial(context.Background(), wsURL)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	l.wss = conn

	roomID := parseInt64(l.ID)
	if err := wsutil.WriteClientBinary(conn, makePushFrame("im_enter_room", encodeImEnterRoom(roomID), 0)); err != nil {
		l.wss.Close()
		l.wss = nil
		return fmt.Errorf("send im_enter_room: %w", err)
	}

	l.wg.Add(2)
	go func() {
		defer l.wg.Done()
		defer close(l.Events)
		defer l.cancel()
		defer func() {
			if l.wss != nil {
				l.wss.Close()
				l.wss = nil
			}
		}()
		l.readSocket()
	}()
	go func() {
		defer l.wg.Done()
		l.sendHeartbeat(roomID)
	}()
	return nil
}

func (l *Live) sendHeartbeat(roomID int64) {
	var seq int64
	ticker := time.NewTicker(wsHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-l.done():
			return
		case <-ticker.C:
			seq++
			payload := makePushFrame("hb", encodeHeartBeat(roomID, seq), 0)
			if err := wsutil.WriteClientBinary(l.wss, payload); err != nil {
				l.t.errHandler(fmt.Errorf("websocket heartbeat failed: %w", err))
				return
			}
		}
	}
}

func (l *Live) readSocket() {
	controlHandler := wsutil.ControlFrameHandler(l.wss, ws.StateClientSide)
	rd := wsutil.Reader{
		Source:         l.wss,
		State:          ws.StateClientSide,
		CheckUTF8:      true,
		OnIntermediate: controlHandler,
	}

	for {
		hdr, err := rd.NextFrame()
		if err != nil {
			l.t.errHandler(fmt.Errorf("websocket read: %w", err))
			return
		}
		if hdr.OpCode.IsControl() {
			if err := controlHandler(hdr, &rd); err != nil {
				l.t.errHandler(fmt.Errorf("websocket control frame: %w", err))
			}
			continue
		}
		if hdr.OpCode == ws.OpClose {
			l.t.warnHandler("websocket closed by server")
			return
		}
		if hdr.OpCode&ws.OpBinary == 0 {
			if _, err := io.Copy(io.Discard, &rd); err != nil {
				return
			}
			continue
		}
		msgBytes, err := io.ReadAll(&rd)
		if err != nil {
			l.t.errHandler(fmt.Errorf("websocket read payload: %w", err))
			return
		}
		if l.parseWssMsg(msgBytes) {
			return // live ended
		}
		select {
		case <-l.done():
			return
		default:
		}
	}
}

// parseWssMsg decodes a WebcastPushFrame; returns true when the live ended.
func (l *Live) parseWssMsg(msg []byte) bool {
	var frame pb.WebcastPushFrame
	if err := proto.Unmarshal(msg, &frame); err != nil {
		l.t.errHandler(fmt.Errorf("unmarshal push frame: %w", err))
		return false
	}
	if frame.PayloadType != "msg" {
		return false
	}

	payload := frame.Payload
	if dec, err := gunzipData(payload); err == nil {
		payload = dec
	}

	var resp pb.WebcastResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		l.t.errHandler(fmt.Errorf("unmarshal webcast response: %w", err))
		return false
	}

	if resp.NeedsAck {
		ack := makePushFrame("ack", resp.InternalExt, frame.LogId)
		_ = wsutil.WriteClientBinary(l.wss, ack)
	}
	l.cursor = resp.Cursor

	ended := false
	for _, rawMsg := range resp.Messages {
		msg, err := parseMsg(rawMsg, l.t.warnHandler, l.t.debugHandler, l.t.enableExperimentalEvents)
		if err != nil || msg == nil {
			continue
		}
		if len(l.Events) == l.chanSize {
			<-l.Events
		}
		l.Events <- msg
		if m, ok := msg.(ControlEvent); ok && (m.Action == 3 || m.Action == 4) {
			ended = true
		}
	}
	return ended
}

func parseInt64(s string) int64 {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int64(c-'0')
	}
	return n
}
