package irc

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Action string

const (
	emptyAction     Action = ""
	ClearChat       Action = "CLEARCHAT"
	ClearMsg        Action = "CLEARMSG"
	GlobalUserState Action = "GLOBALUSERSTATE"
	Join            Action = "JOIN"
	Part            Action = "PART"
	PrivMsg         Action = "PRIVMSG"
	RoomState       Action = "ROOMSTATE"
	UserNotice      Action = "USERNOTICE"
	UserState       Action = "USERSTATE"
)

type Size int

const (
	_ Size = iota
	Small
	Medium
	Large
)

func (s Size) String() string {
	return fmt.Sprintf("%.01f", float64(s))
}

type Emote struct {
	ID       int
	StartIdx int
	EndIdx   int
	Text     string
	raw      string
}

func newEmote(e, msg string) Emote {
	parts := strings.Split(e, ":")
	if len(parts) != 2 {
		log.Errorf("Unexpected parts length: %q", parts)
		return Emote{}
	}
	indexes := strings.Split(parts[1], "-")

	emote := Emote{raw: e}
	emote.ID, _ = strconv.Atoi(parts[0])
	emote.StartIdx, _ = strconv.Atoi(indexes[0])
	emote.EndIdx, _ = strconv.Atoi(indexes[1])
	emote.Text = msg[emote.StartIdx : emote.EndIdx+1]

	return emote
}

func (e Emote) URL(size Size) string {
	return fmt.Sprintf("https://static-cdn.jtvnw.net/emoticons/v1/%d/%s", e.ID, size)
}

type Event struct {
	Action  Action
	Raw     string
	Tags    map[string]string
	Message string
	Channel string
	Nick    string
	User    string
	Host    string
}

func (e Event) ReifyTags() interface{} {
	tags := e.Tags
	if len(tags) == 0 {
		return nil
	}

	var t taggable
	switch e.Action {
	case ClearChat:
		t = &ClearChatTags{}
	case ClearMsg:
		t = &ClearMsgTags{}
	case GlobalUserState:
		t = &GlobalUserStateTags{}
	case PrivMsg:
		t = &PrivMsgTags{}
	case RoomState:
		t = &RoomStateTags{}
	// TODO
	// case UserNotice:
	//     t = &UserNoticeTags{}
	case UserState:
		t = &UserStateTags{}
	default:
		return nil
	}

	t.newFromTags(tags, e.Message)
	return t
}

type taggable interface {
	newFromTags(map[string]string, string) error
}

type ClearChatTags struct {
	BanDuration time.Duration
}

func (cct *ClearChatTags) newFromTags(tags map[string]string, msg string) error {
	v, err := strconv.Atoi(tags["ban-duration"])
	if err != nil {
		return err
	}
	cct.BanDuration = time.Duration(v) * time.Second
	return nil
}

type ClearMsgTags struct {
	Login           string
	Message         string
	TargetMessageID uuid.UUID
}

func (cmt *ClearMsgTags) newFromTags(tags map[string]string, msg string) error {
	cmt.Login = tags["login"]
	cmt.Message = tags["message"]
	cmt.TargetMessageID, _ = uuid.Parse(tags["target-msg-id"])
	return nil
}

type GlobalUserStateTags struct {
	BadgeInfo   map[string]int
	Badges      map[string]int
	Color       string
	DisplayName string
	EmoteSets   []int
	UserID      int
}

func (gust *GlobalUserStateTags) newFromTags(tags map[string]string, msg string) error {
	gust.BadgeInfo = make(map[string]int)
	parseCsvSlash(tags["badge-info"], gust.BadgeInfo)
	gust.Badges = make(map[string]int)
	parseCsvSlash(tags["badges"], gust.Badges)
	gust.Color = tags["color"]
	gust.DisplayName = tags["display-name"]
	gust.EmoteSets = []int{}
	for _, e := range strings.Split(tags["emote-sets"], ",") {
		v, _ := strconv.Atoi(e)
		gust.EmoteSets = append(gust.EmoteSets, v)
	}
	gust.UserID, _ = strconv.Atoi(tags["user-id"])

	return nil
}

type UserStateTags struct {
	*GlobalUserStateTags
	Mod bool
}

func (ust *UserStateTags) newFromTags(tags map[string]string, msg string) error {
	ust.GlobalUserStateTags = &GlobalUserStateTags{}
	ust.GlobalUserStateTags.newFromTags(tags, msg)
	ust.Mod = tags["mod"] == "1"

	return nil
}

type PrivMsgTags struct {
	*UserStateTags
	Bits      int
	Emotes    []Emote
	ID        uuid.UUID
	RoomID    int
	Timestamp time.Time
	UserID    int
}

func (pmt *PrivMsgTags) newFromTags(tags map[string]string, msg string) error {
	pmt.UserStateTags = &UserStateTags{}
	pmt.UserStateTags.newFromTags(tags, msg)
	if b, ok := tags["bits"]; ok {
		pmt.Bits, _ = strconv.Atoi(b)
	}

	pmt.Emotes = []Emote{}
	for _, e := range strings.Split(tags["emotes"], "/") {
		if e == "" {
			continue
		}
		pmt.Emotes = append(pmt.Emotes, newEmote(e, msg))
	}

	pmt.ID, _ = uuid.Parse(tags["id"])
	pmt.RoomID, _ = strconv.Atoi(tags["room-id"])

	ts, _ := strconv.Atoi(tags["tmi-sent-ts"])
	pmt.Timestamp = time.Unix(0, int64(ts)*1000*1000) // given in millis, let's not lose resolution
	pmt.UserID, _ = strconv.Atoi(tags["user-id"])

	return nil
}

type RoomStateTags struct {
	EmoteOnly     bool
	FollowersOnly int
	R9K           bool
	Slow          int
	SubsOnly      bool
}

func (rst *RoomStateTags) newFromTags(tags map[string]string, msg string) error {
	rst.EmoteOnly = tags["emote-only"] == "1"
	rst.FollowersOnly, _ = strconv.Atoi(tags["followers-only"])
	rst.R9K = tags["r9k"] == "1"
	rst.Slow, _ = strconv.Atoi(tags["slow"])
	rst.SubsOnly = tags["subs-only"] == "1"

	return nil
}

// type UserNoticeTags struct {
// }

// func (unt *UserNoticeTags) newFromTags(tags map[string]string) error {
// }

func parseCsvSlash(v string, target map[string]int) {
	if v == "" {
		return
	}

	for _, entry := range strings.Split(v, ",") {
		parts := strings.Split(entry, "/")
		v, _ := strconv.Atoi(parts[1])
		target[parts[0]] = v
	}
}
