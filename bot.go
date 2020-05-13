package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/alpinehq/twitch/irc"
)

type Bot interface {
	OnPrivMsg(irc.Event, *irc.Conn)
}

// BotAdapter implements nops for all callback functions for any
// bot that only wants to handle a subset of actions
type BotAdapter struct{}

func (ba *BotAdapter) OnPrivMsg(e irc.Event, c *irc.Conn) {}

type PingBot struct {
	BotAdapter
}

func (p *PingBot) OnPrivMsg(e irc.Event, c *irc.Conn) {
	if strings.HasPrefix(e.Message, "!ping") {
		c.Send(irc.PrivateMessage{
			Channel: e.Channel,
			Message: fmt.Sprintf("@%s pong", e.User),
		})
	}
}

// shameless ripoff from twitch.tv/blendedsoftware
type LightBot struct {
	BotAdapter

	colorRe *regexp.Regexp
}

func NewLightBot() Bot {
	return &LightBot{
		colorRe: regexp.MustCompile(`(?i)(#[a-f0-9]{6}|[a-z]+)$`),
	}
}

func (l *LightBot) OnPrivMsg(e irc.Event, c *irc.Conn) {
	if !strings.HasPrefix(e.Message, "!lights") {
		return
	}

	// TODO: allow user's to alias `!lights alias <name> #color`

	match := l.colorRe.FindString(strings.TrimPrefix(e.Message, "!lights "))
	if len(match) < 1 {
		return
	}

	if strings.HasPrefix(match, "#") {
		c.Send(irc.PrivateMessage{
			Channel: e.Channel,
			Message: fmt.Sprintf("%s set the lights to %s", e.User, match),
		})
	} else {
		c.Send(irc.PrivateMessage{
			Channel: e.Channel,
			Message: fmt.Sprintf("@%s Sorry, I don't know that color", e.User),
		})
	}
}

type EmoteBot struct {
	cacheDir         string
	latestEmotePath  string
	latestEmoterPath string
}

func NewEmoteBot() Bot {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatalf("Unable to get user's cache directory: %s", err)
	}

	cacheDir = filepath.Join(cacheDir, "twitch_emotes")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Fatalf("Unable to create cache directory: %s", err)
	}

	return &EmoteBot{
		cacheDir:         cacheDir,
		latestEmotePath:  filepath.Join(cacheDir, "latest_emote.png"),
		latestEmoterPath: filepath.Join(cacheDir, "latest_emoter.txt"),
	}
}

func (eb *EmoteBot) OnPrivMsg(e irc.Event, c *irc.Conn) {
	t := e.ReifyTags()
	pm, ok := t.(*irc.PrivMsgTags)
	if !ok || len(pm.Emotes) == 0 {
		return
	}

	// download the images
	seenEmotes := map[int]bool{}
	for _, emote := range pm.Emotes {
		fmt.Println(emote.Text)
		if seenEmotes[emote.ID] {
			continue
		}
		seenEmotes[emote.ID] = true
		emotePath := filepath.Join(eb.cacheDir, fmt.Sprintf("%d_%s", emote.ID, irc.Large)+".png")

		if _, err := os.Stat(emotePath); !os.IsNotExist(err) {
			continue
		}

		resp, err := http.Get(emote.URL(irc.Large))
		if err != nil {
			log.Errorf("Unable to get emote %d: %s", emote.ID, err)
			continue
		}
		defer resp.Body.Close()

		f, err := os.Create(emotePath)
		if err != nil {
			log.Errorf("Unable to create emote path: %s", err)
			continue
		}

		io.Copy(f, resp.Body)
	}

	// delete the old symlink
	// if _, err := os.Lstat(eb.latestEmotePath); !os.IsNotExist(err) {
	//     if err := os.Remove(eb.latestEmotePath); err != nil {
	//         log.Errorf("Unable to remove latest emote path: %s", err)
	//         return
	//     }
	// }

	// create new softlink
	// if err := os.Symlink(emotePath, eh.latestEmotePath); err != nil {
	//     return fmt.Errorf("unable to symlink new emote: %w", err)
	// }

	// write emoter text
	// u := e.User
	// if pm.DisplayName != "" {
	//     u = pm.DisplayName
	// }
	// if err := ioutil.WriteFile(eb.latestEmoterPath, []byte(u), 0644); err != nil {
	//     log.Errorf("Unable to write new emoter: %s", err)
	// }
}
