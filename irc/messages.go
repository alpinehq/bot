package irc

import "fmt"

type Sendable interface {
	fmt.Stringer
}

type ChannelJoin struct {
	Channel string
}

func (cj ChannelJoin) String() string {
	return fmt.Sprintf("JOIN #%s", cj.Channel)
}

type ChannelPart struct {
	Channel string
}

func (cp ChannelPart) String() string {
	return fmt.Sprintf("PART #%s", cp.Channel)
}

type PrivateMessage struct {
	Channel string
	Message string
}

func (pm PrivateMessage) String() string {
	return fmt.Sprintf("PRIVMSG #%s :%s", pm.Channel, pm.Message)
}

type Capability string

const (
	MembershipCapability Capability = "membership"
	TagCapability        Capability = "tags"
	CommandCapability    Capability = "commands"
)

type CapabilityRequest struct {
	Capability Capability
}

func (cr CapabilityRequest) String() string {
	return fmt.Sprintf("CAP REQ :twitch.tv/%s", string(cr.Capability))
}

type Password struct {
	Password string
}

func (p Password) String() string {
	return fmt.Sprintf("PASS %s", p.Password)
}

type Username struct {
	Username string
}

func (p Username) String() string {
	return fmt.Sprintf("NICK %s", p.Username)
}

type Pong struct{}

func (p Pong) String() string {
	return "PONG :tmi.twitch.tv"
}
