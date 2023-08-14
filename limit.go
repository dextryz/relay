package main

/// "max_message_length": 16384,
/// "max_subscriptions": 20,
/// "max_filters": 100,
/// "max_limit": 5000,
/// "max_subid_length": 100,
/// "min_prefix": 4,
/// "max_event_tags": 100,
/// "max_content_length": 8196,
/// "min_pow_difficulty": 30,
/// "auth_required": true,
/// "payment_required": true,
import "fmt"

var ErrMaxLimit = fmt.Errorf("max limit of events or authors specified")

// Relay limitations from NIP-11
type limit struct {
	max_limit   int
	max_filters int
}
