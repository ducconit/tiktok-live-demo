package live

import (
	"fmt"
	"log/slog"
	"time"

	gotiktoklive "github.com/steampoweredtaco/gotiktoklive"
)

// event — event chuẩn publish lên Sockudo channel "user_<username>" (event "event").
type event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
	Ts   int64  `json:"ts"`
}

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
	return event{Type: "status", Data: map[string]any{"state": state}, Ts: time.Now().UnixMilli()}
}

// roomPreview returns a lightweight preview (live status + room info) for a
// username, used by the frontend's TanStack Query before connecting.
func roomPreview(username string) (map[string]any, error) {
	t, err := gotiktoklive.NewTikTok()
	if err != nil {
		return nil, err
	}
	info, err := t.GetRoomInfo(username)
	if err != nil {
		// Offline / not found → not live.
		return map[string]any{"live": false}, nil
	}
	out := map[string]any{
		"live":      true,
		"title":     info.Title,
		"userCount": info.UserCount,
	}
	if info.Owner != nil {
		owner := map[string]any{
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

// startLive — track một username: connect TikTok, publish mọi event lên Sockudo
// channel "user_<username>". Trả controller (Stop) + connected status event.
func (s *Service) startLive(username string) (controller, event, error) {
	ss, err := s.getSigner()
	if err != nil {
		return nil, event{}, fmt.Errorf("self-hosted signer init: %w", err)
	}

	opts := []gotiktoklive.TikTokLiveOption{
		gotiktoklive.SigningFunc(ss.signFetch),
		gotiktoklive.SigningURLFunc(ss.signOnly),
		gotiktoklive.PollInterval(time.Duration(s.cfg.PollIntervalMs) * time.Millisecond),
	}
	if s.cfg.ConnectionMode == "websocket" {
		opts = append(opts, gotiktoklive.WebSocketMode())
	}

	t, err := gotiktoklive.NewTikTok(opts...)
	if err != nil {
		return nil, event{}, err
	}
	t.SetErrorHandler(func(v ...interface{}) {
		slog.Error("tiktok error", "err", v)
	})
	t.SetInfoHandler(func(v ...interface{}) {
		slog.Info("tiktok info", "info", v)
	})

	live, err := t.TrackUser(username)
	if err != nil {
		return nil, event{}, err
	}

	if s.logger != nil {
		s.logger.line(fmt.Sprintf("connected username=%s roomId=%s", username, live.ID))
	}

	connected := event{
		Type: "status",
		Data: map[string]any{
			"state":    "connected",
			"roomId":   live.ID,
			"roomInfo": roomInfoToJSON(live.Info),
		},
		Ts: time.Now().UnixMilli(),
	}

	c := &liveController{live: live}
	go func() {
		for ev := range live.Events {
			if s.logger != nil {
				s.logger.raw(ev)
			}
			if ev.IsHistory() {
				continue
			}
			if e, ok := toEvent(ev); ok {
				if err := s.pub.publish("user_"+username, "event", e); err != nil {
					slog.Error("sockudo publish", "err", err, "channel", "user_"+username)
				}
			}
		}
		if s.logger != nil {
			s.logger.line("disconnected (events channel closed)")
		}
	}()
	return c, connected, nil
}

// toEvent — map gotiktoklive event → event JSON publish lên Sockudo.
func toEvent(ev gotiktoklive.Event) (event, bool) {
	now := time.Now().UnixMilli()
	switch e := ev.(type) {
	case gotiktoklive.ChatEvent:
		return event{Type: "chat", Data: map[string]any{
			"comment":   e.Comment,
			"user":      userToJSON(e.User),
			"msgId":     e.MessageID,
			"timestamp": e.Timestamp,
		}, Ts: now}, true

	case gotiktoklive.UserEvent:
		t := "member"
		switch e.Event {
		case gotiktoklive.USER_FOLLOW:
			t = "follow"
		case gotiktoklive.USER_SHARE:
			t = "share"
		}
		data := map[string]any{"user": userToJSON(e.User)}
		if t == "member" && e.MemberCount > 0 {
			data["memberCount"] = e.MemberCount
		}
		return event{Type: t, Data: data, Ts: now}, true

	case gotiktoklive.GiftEvent:
		return event{Type: "gift", Data: map[string]any{
			"giftId":       e.ID,
			"giftType":     e.Type,
			"repeatCount":  e.RepeatCount,
			"repeatEnd":    e.RepeatEnd,
			"giftName":     e.Name,
			"diamondCount": e.Diamonds,
			"user":         userToJSON(e.User),
			"toUserId":     e.ToUserID,
		}, Ts: now}, true

	case gotiktoklive.LikeEvent:
		return event{Type: "like", Data: map[string]any{
			"likeCount":      e.Likes,
			"totalLikeCount": e.TotalLikes,
			"user":           userToJSON(e.User),
		}, Ts: now}, true

	case gotiktoklive.ViewersEvent:
		return event{Type: "roomUser", Data: map[string]any{"viewerCount": e.Viewers}, Ts: now}, true

	case gotiktoklive.QuestionEvent:
		return event{Type: "questionNew", Data: map[string]any{
			"questionText": e.Quesion,
			"user":         userToJSON(e.User),
		}, Ts: now}, true

	case gotiktoklive.ControlEvent:
		if e.Action == 3 {
			return statusEvent("ended"), true
		}
		return event{}, false

	case gotiktoklive.IntroEvent:
		return event{Type: "liveIntro", Data: map[string]any{
			"title": e.Title,
			"user":  userToJSON(e.User),
		}, Ts: now}, true

	case gotiktoklive.MicBattleEvent:
		users := make([]userJSON, 0, len(e.Users))
		for _, u := range e.Users {
			users = append(users, userToJSON(u))
		}
		return event{Type: "linkMicBattle", Data: map[string]any{"users": users}, Ts: now}, true

	case gotiktoklive.BattlesEvent:
		return event{Type: "linkMicArmies", Data: map[string]any{
			"status":  e.Status,
			"battles": battlesToJSON(e.Battles),
		}, Ts: now}, true

	case gotiktoklive.DisconnectEvent:
		return statusEvent("disconnected"), true
	}
	return event{}, false
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
