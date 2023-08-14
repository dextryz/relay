package main

// https://github.com/nostr-protocol/nips/blob/vending-machine/11.md

import "fmt"

var ErrMaxLimit = fmt.Errorf("max limit of events or authors specified")

// Relay limitations from NIP-11
type limit struct {
	max_limit   int
	max_filters int
}
