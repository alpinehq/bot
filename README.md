# bot

This is a raw bot for twitch, manually handling the IRC protocol loosely - though enough to support [twitch](https://twitch.tv).

## Usage

Right now this is built for my interests, however it is trivial to extend. You can add your "bots" (which are just callback handlers) to the bots slice in main.go. See bots.go for examples.

Otherwise, if you just want to run it:

```shell
$ go run $(ls *.go | grep -v "_test.go") run --username=YOUR_USERNAME --password=YOUR_PASSWORD --channels=YOUR_CHANNEL
```
