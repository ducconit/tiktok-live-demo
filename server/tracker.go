package main

import (
	"fmt"
	"sync"
	"time"

	gotiktoklive "github.com/steampoweredtaco/gotiktoklive"
)

type event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
	Ts   int64       `json:"ts"`
}

type emitFunc func(event)

type controller interface {
	Stop()
}

type liveController struct {
	live *gotiktoklive.Live
}

func (c *liveController) Stop() {
	c.live.Close()
}

type userJSON struct {
	UserID            int64  `json:"userId"`
	UniqueID          string `json:"uniqueId"`
	Nickname          string `json:"nickname"`
	ProfilePictureURL string `json:"profilePictureUrl"`
}

func userToJSON(u *gotiktoklive.User) userJSON {
	if u == nil {
		return userJSON{}
	}
	j := userJSON{
		UserID:   u.ID,
		UniqueID: u.Username,
		Nickname: u.Nickname,
	}
	if u.ProfilePicture != nil && len(u.ProfilePicture.Urls) > 0 {
		j.ProfilePictureURL = u.ProfilePicture.Urls[0]
	}
	return j
}

type hostJSON struct {
	UniqueID          string `json:"uniqueId"`
	Nickname          string `json:"nickname"`
	ProfilePictureURL string `json:"profilePictureUrl"`
}

type roomInfoJSON struct {
	Title     string    `json:"title"`
	UserCount int       `json:"userCount"`
	Owner     *hostJSON `json:"owner"`
}

func roomInfoToJSON(info *gotiktoklive.RoomInfo) *roomInfoJSON {
	if info == nil {
		return nil
	}
	j := &roomInfoJSON{
		Title:     info.Title,
		UserCount: info.UserCount,
	}
	if info.Owner != nil {
		host := &hostJSON{
			UniqueID: info.Owner.Username,
			Nickname: info.Owner.Nickname,
		}
		avatars := info.Owner.AvatarThumb.URLList
		if len(avatars) == 0 {
			avatars = info.Owner.AvatarMedium.URLList
		}
		if len(avatars) > 0 {
			host.ProfilePictureURL = avatars[0]
		}
		j.Owner = host
	}
	return j
}

func statusEvent(state string) event {
	return event{Type: "status", Data: map[string]interface{}{"state": state}, Ts: time.Now().UnixMilli()}
}

// roomPreview returns a lightweight preview (live status + room info) for a
// username, used by the frontend's TanStack Query before connecting.
func roomPreview(username string) (map[string]interface{}, error) {
	t, err := gotiktoklive.NewTikTok()
	if err != nil {
		return nil, err
	}
	info, err := t.GetRoomInfo(username)
	if err != nil {
		// Offline / not found → not live.
		return map[string]interface{}{"live": false}, nil
	}
	out := map[string]interface{}{
		"live":      true,
		"title":     info.Title,
		"userCount": info.UserCount,
	}
	if info.Owner != nil {
		owner := map[string]interface{}{
			"uniqueId": info.Owner.Username,
			"nickname": info.Owner.Nickname,
		}
		if avatars := info.Owner.AvatarThumb.URLList; len(avatars) > 0 {
			owner["profilePictureUrl"] = avatars[0]
		}
		out["owner"] = owner
	}
	return out, nil
}

func startTracker(username string, emit emitFunc, cfg config) (controller, error) {
	return startLive(username, emit, cfg)
}

// getSelfSigner lazily initializes the shared self-hosted signer (QuickJS +
// Chrome TLS). QuickJS init is expensive (~seconds), so it is reused across
// trackers.
var (
	selfSignerOnce sync.Once
	selfSignerInst *selfSigner
	selfSignerErr  error
)

func getSelfSigner() (*selfSigner, error) {
	selfSignerOnce.Do(func() {
		selfSignerInst, selfSignerErr = newSelfSigner()
	})
	return selfSignerInst, selfSignerErr
}

func startLive(username string, emit emitFunc, cfg config) (controller, error) {
	opts := []gotiktoklive.TikTokLiveOption{}

	// Self-hosted signer (QuickJS) — no third-party dependency.
	ss, err := getSelfSigner()
	if err != nil {
		return nil, fmt.Errorf("self-hosted signer init: %w", err)
	}
	opts = append(opts, gotiktoklive.SigningFunc(ss.signFetch), gotiktoklive.SigningURLFunc(ss.signOnly))

	// Connection mode + polling interval (configurable via CONNECTION_MODE /
	// POLL_INTERVAL_MS). Mặc định long-poll; websocket fallback long-poll nếu lỗi.
	opts = append(opts, gotiktoklive.PollInterval(time.Duration(cfg.PollIntervalMs)*time.Millisecond))
	switch cfg.ConnectionMode {
	case "websocket":
		logf("[tiktok-bar] connection mode: websocket")
		opts = append(opts, gotiktoklive.WebSocketMode())
	default:
		logf("[tiktok-bar] connection mode: long_poll (poll %dms)", cfg.PollIntervalMs)
	}

	t, err := gotiktoklive.NewTikTok(opts...)
	if err != nil {
		return nil, err
	}
	t.SetErrorHandler(func(v ...interface{}) {
		logf("tiktok error: %v", v)
	})
	t.SetInfoHandler(func(v ...interface{}) {
		logf("tiktok info: %v", v)
	})

	live, err := t.TrackUser(username)
	if err != nil {
		return nil, err
	}

	if rawLogger != nil {
		rawLogger.line(fmt.Sprintf("connected username=%s roomId=%s", username, live.ID))
	}

	emit(event{
		Type: "status",
		Data: map[string]interface{}{
			"state":    "connected",
			"roomId":   live.ID,
			"roomInfo": roomInfoToJSON(live.Info),
		},
		Ts: time.Now().UnixMilli(),
	})

	c := &liveController{live: live}
	go func() {
		for ev := range live.Events {
			if rawLogger != nil {
				rawLogger.raw(ev)
			}
			if ev.IsHistory() {
				continue
			}
			relayEvent(ev, emit)
		}
		if rawLogger != nil {
			rawLogger.line("disconnected (events channel closed)")
		}
	}()
	return c, nil
}

func relayEvent(ev gotiktoklive.Event, emit emitFunc) {
	now := time.Now().UnixMilli()
	switch e := ev.(type) {
	case gotiktoklive.ChatEvent:
		emit(event{Type: "chat", Data: map[string]interface{}{
			"comment":   e.Comment,
			"user":      userToJSON(e.User),
			"msgId":     e.MessageID,
			"timestamp": e.Timestamp,
		}, Ts: now})

	case gotiktoklive.UserEvent:
		t := "member"
		switch e.Event {
		case gotiktoklive.USER_FOLLOW:
			t = "follow"
		case gotiktoklive.USER_SHARE:
			t = "share"
		}
		data := map[string]interface{}{"user": userToJSON(e.User)}
		if t == "member" && e.MemberCount > 0 {
			data["memberCount"] = e.MemberCount
		}
		emit(event{Type: t, Data: data, Ts: now})

	case gotiktoklive.GiftEvent:
		emit(event{Type: "gift", Data: map[string]interface{}{
			"giftId":       e.ID,
			"giftType":     e.Type,
			"repeatCount":  e.RepeatCount,
			"repeatEnd":    e.RepeatEnd,
			"giftName":     e.Name,
			"diamondCount": e.Diamonds,
			"user":         userToJSON(e.User),
			"toUserId":     e.ToUserID,
		}, Ts: now})

	case gotiktoklive.LikeEvent:
		emit(event{Type: "like", Data: map[string]interface{}{
			"likeCount":      e.Likes,
			"totalLikeCount": e.TotalLikes,
			"user":           userToJSON(e.User),
		}, Ts: now})

	case gotiktoklive.ViewersEvent:
		emit(event{Type: "roomUser", Data: map[string]interface{}{"viewerCount": e.Viewers}, Ts: now})

	case gotiktoklive.QuestionEvent:
		emit(event{Type: "questionNew", Data: map[string]interface{}{
			"questionText": e.Quesion,
			"user":         userToJSON(e.User),
		}, Ts: now})

	case gotiktoklive.ControlEvent:
		if e.Action == 3 {
			emit(statusEvent("ended"))
		}

	case gotiktoklive.IntroEvent:
		emit(event{Type: "liveIntro", Data: map[string]interface{}{
			"title": e.Title,
			"user":  userToJSON(e.User),
		}, Ts: now})

	case gotiktoklive.MicBattleEvent:
		users := make([]userJSON, 0, len(e.Users))
		for _, u := range e.Users {
			users = append(users, userToJSON(u))
		}
		emit(event{Type: "linkMicBattle", Data: map[string]interface{}{"users": users}, Ts: now})

	case gotiktoklive.BattlesEvent:
		emit(event{Type: "linkMicArmies", Data: map[string]interface{}{
			"status":  e.Status,
			"battles": battlesToJSON(e.Battles),
		}, Ts: now})

	case gotiktoklive.DisconnectEvent:
		emit(statusEvent("disconnected"))
	}
}

type battleJSON struct {
	Host   int64             `json:"host"`
	Groups []battleGroupJSON `json:"groups"`
}

type battleGroupJSON struct {
	Points int        `json:"points"`
	Users  []userJSON `json:"users"`
}

func battlesToJSON(battles []*gotiktoklive.Battle) []battleJSON {
	out := make([]battleJSON, 0, len(battles))
	for _, b := range battles {
		if b == nil {
			continue
		}
		bj := battleJSON{Host: b.Host}
		for _, g := range b.Groups {
			if g == nil {
				continue
			}
			users := make([]userJSON, 0, len(g.Users))
			for _, u := range g.Users {
				users = append(users, userToJSON(u))
			}
			bj.Groups = append(bj.Groups, battleGroupJSON{Points: g.Points, Users: users})
		}
		out = append(out, bj)
	}
	return out
}
