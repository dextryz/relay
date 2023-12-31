package main

import "github.com/ffiat/nostr"

// 1. Client manages the filters.

type client struct {

	// NIP-01: Send events received from broadcaster to client.
	send chan nostr.MessageEvent

	// NIP-20: Result channel for event published.
	result chan nostr.MessageOk
}

func (s *client) Close() {
	close(s.send)
	close(s.result)
}
