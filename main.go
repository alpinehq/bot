package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/alpinehq/twitch/irc"
)

func main() {
	app := &cli.App{
		Name: "twitch",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				EnvVars: []string{"TWITCH_LOG_LEVEL"},
				Value:   "info",
			},
		},
		Before: func(c *cli.Context) error {
			log.SetFormatter(&log.TextFormatter{
				FullTimestamp:   true,
				TimestampFormat: time.RFC3339,
				ForceQuote:      true,
			})
			log.SetOutput(os.Stdout)
			if level, err := log.ParseLevel(c.String("log-level")); err == nil {
				log.SetLevel(level)
			} else {
				return err
			}
			return nil
		},
		Commands: cli.Commands{
			{
				Name: "run",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "username",
						EnvVars:  []string{"TWITCH_BOT_USERNAME"},
						Required: true,
					},
					&cli.StringFlag{
						Name:     "password",
						EnvVars:  []string{"TWITCH_BOT_PASSWORD"},
						Required: true,
					},
					&cli.DurationFlag{
						Name:    "wait",
						EnvVars: []string{"TWITCH_SHUTDOWN_WAIT"},
						Value:   30 * time.Second,
					},
					&cli.StringSliceFlag{
						Name:  "channels",
						Value: cli.NewStringSlice("alpinehq_"),
					},
				},
				Action: func(c *cli.Context) error {
					conn := irc.New("irc.chat.twitch.tv:6697", c.String("username"), c.String("password"))

					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					wg := &sync.WaitGroup{}
					wg.Add(1)
					go func() {
						defer wg.Done()
						conn.Run(ctx)
					}()

					conn.WaitForLogin()

					for _, channel := range c.StringSlice("channels") {
						log.Debugf("Joining channel %s", channel)
						conn.Send(irc.ChannelJoin{Channel: channel})
					}

					wg.Add(1)
					go func() {
						defer wg.Done()
						bots := []Bot{
							// &PingBot{},
							// NewLightBot(),
							// NewEmoteBot(),
						}

						for {
							select {
							case event := <-conn.Events():
								log.Debugf("Event: %+v", event)

								switch event.Action {
								case irc.PrivMsg:
									for _, b := range bots {
										b.OnPrivMsg(event, conn)
									}
								default:
									log.Debugf("Ignored event type: %+v", event)
								}

							case <-ctx.Done():
								return
							}
						}
					}()

					log.Info("Listening...")
					stop := make(chan os.Signal, 1)
					signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
					<-stop
					log.Info("Shutdown signal received...")

					// Graceful shutdown
					shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), c.Duration("wait"))
					go func() {
						defer shutdownCancel()
						cancel()
						wg.Wait()
					}()

					<-shutdownCtx.Done()
					if shutdownCtx.Err() == context.DeadlineExceeded {
						return errors.New("graceful shutdown timeout elapsed")
					}
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
