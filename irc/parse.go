package irc

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	matchRe = regexp.MustCompile(`:(?:(?P<nick>[A-z0-9]+)!(?P<user>[A-z0-9]+)@(?:[A-z0-9]+)\.)?(?P<server>[^ ]+) (?P<action>[A-Z0-9]+(?: \* [A-Z]+)?)(?: #?(?P<channel>[A-z]+))?(?: :(?P<message>.*))?`)
)

func parse(line string) Event {
	event := Event{
		Tags: make(map[string]string),
		Raw:  line,
	}

	// Parse the message tags, which are an optional prefix - effectively.
	if strings.HasPrefix(line, "@") {
		line = strings.TrimPrefix(line, "@")
		parts := strings.SplitN(line, " ", 2)
		line = parts[1]
		for _, kv := range strings.Split(parts[0], ";") {
			nameVal := strings.Split(kv, "=")
			name := nameVal[0]
			val := ""
			if len(nameVal) > 1 {
				val = nameVal[1]
			}

			event.Tags[name] = val
		}
	}

	// Parse the message
	if strings.HasPrefix(line, ":") {
		match := matchRe.FindStringSubmatch(line)
		if len(match) > 0 {
			event.Nick = match[1]
			event.User = match[2]
			event.Host = match[3]
			event.Action = Action(match[4])
			event.Channel = match[5]
			event.Message = match[6]
		}
		// TODO parse the join messages like 353, 36, etc.
	}

	if event.Action == emptyAction {
		log.Debugf("Unhandled action: %s", event.Raw)
	}

	return event
}
