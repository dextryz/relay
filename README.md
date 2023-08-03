# Tiny Relay

A tiny minimal nostr relay in Go.

## DONE

- Register connected clients via Hub-and-Spoke pattern.
- Receive a NIP-01 text event and respond with a NIP-20 OK.

## TODO

- Store event in database.
- Broadcast received events to registered clients.

## Usage

```shell
# Build binary
go build

# Run relay
: ./relay "localhost:8000"
```
