package irc

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := map[string]struct {
		result Event
		line   string
	}{
		"privmsg": {
			line: ":alpinesoftware!alpinesoftware@alpinesoftware.tmi.twitch.tv PRIVMSG #blendedsoftware :hello world",
			result: Event{
				Action:  PrivMsg,
				Channel: "blendedsoftware",
				Host:    "tmi.twitch.tv",
				Message: "hello world",
				Nick:    "alpinesoftware",
				User:    "alpinesoftware",
				Tags:    make(map[string]string),
			},
		},
		"privmsg_tags": {
			line: "@badge-info=;badges=;color=;display-name=nerdwaller_bot;emote-sets=0;mod=0;subscriber=0;user-type= :alpinesoftware!alpinesoftware@alpinesoftware.tmi.twitch.tv PRIVMSG #blendedsoftware :hello world",
			result: Event{
				Action:  PrivMsg,
				Channel: "blendedsoftware",
				Host:    "tmi.twitch.tv",
				Message: "hello world",
				Nick:    "alpinesoftware",
				User:    "alpinesoftware",
				Tags: map[string]string{
					"badge-info":   "",
					"badges":       "",
					"color":        "",
					"display-name": "nerdwaller_bot",
					"emote-sets":   "0",
					"mod":          "0",
					"subscriber":   "0",
					"user-type":    "",
				},
			},
			"": {
				line: ":tmi.twitch.tv 001 nerdwaller_bot :Welcome, GLHF!",
			},
		},
	}

	for name, tc := range cases {
		tc.result.Raw = tc.line
		r := parse(tc.line)
		if !reflect.DeepEqual(r, tc.result) {
			t.Errorf("[%s] Parse result '%+v' != expected '%+v'", name, r, tc.result)
		}
	}
}
