package irc

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type backoff struct {
	def     time.Duration
	max     time.Duration
	current time.Duration
	factor  int64
}

func (b *backoff) reset() {
	b.current = b.def
}

func (b *backoff) wait() {
	time.Sleep(b.current)
	next := time.Duration((b.current.Milliseconds() * b.factor)) * time.Millisecond
	if next > b.max {
		b.current = b.max
	} else {
		b.current = next
	}
}

type event struct {
	line string
	err  error
}

type Conn struct {
	host     string
	username string
	password string

	events   chan Event
	writer   chan string
	reader   chan event
	loggedIn bool

	rw *bufio.ReadWriter
}

func New(host, username, password string) *Conn {
	return &Conn{
		host:     host,
		username: username,
		password: password,
		events:   make(chan Event, 100),
		writer:   make(chan string, 10),
		reader:   make(chan event, 10),
	}
}

func (c *Conn) Events() <-chan Event {
	return c.events
}

func (c *Conn) WaitForLogin() {
	for {
		if c.loggedIn {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (c *Conn) Run(ctx context.Context) {
	backer := backoff{
		def:    time.Second * 1,
		max:    time.Second * 30,
		factor: 2,
	}

	go c.writerLoop(ctx)
	go c.readerLoop(ctx)

RUNNER:
	c.loggedIn = false
	conn, err := tls.DialWithDialer(&net.Dialer{
		Timeout: 15 * time.Second,
	}, "tcp", c.host, &tls.Config{})
	if err != nil {
		log.Errorf("Error connecting, backing off: %s", err)
		backer.wait()
		goto RUNNER
	}
	defer conn.Close()
	backer.reset()

	c.rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	c.Send(CapabilityRequest{MembershipCapability})
	c.Send(CapabilityRequest{TagCapability})
	c.Send(CapabilityRequest{CommandCapability})

	c.Send(Password{c.password})
	c.Send(Username{c.username})

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-c.reader:
			if event.err != nil {
				goto RUNNER
			}
			line := strings.TrimSpace(event.line)
			if !c.loggedIn && strings.Contains(line, ":Welcome, GLHF!") {
				c.loggedIn = true
			}

			if strings.HasPrefix(line, "PING") {
				log.Debug("Received PING, responding with PONG")
				c.Send(Pong{})
				continue
			}

			select {
			case c.events <- parse(line):
			default:
				log.Warnf("Dropping message due to blocked channel: %s", line)
			}
		}
	}
}

func (c *Conn) writerLoop(ctx context.Context) {
	for {
		select {
		case msg := <-c.writer:
			if c.rw != nil {
				c.rw.WriteString(msg + "\r\n")
				c.rw.Flush()
			} else {
				log.Warnf("Unable to write message, dropping: %s", msg)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Conn) readerLoop(ctx context.Context) {
	for {
		if c.rw == nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		line, err := c.rw.ReadString('\n')

		select {
		case <-ctx.Done():
			return
		case c.reader <- event{line, err}:
		}
	}
}

func (c *Conn) Send(s Sendable) {
	c.writer <- s.String()
}
