# Tiny Relay

A tiny nostr relay implemented in Go with an embedded SQLite database.

## Todo

- [ ] Broadcast received events to registered clients.
- [ ] Implement structure logging.
- [X] Handle NIP-11 requests for relay information.
- [X] Aggregate filter query to SQLite.
- [X] Send stored event to client REQ via event ID.
- [X] Send stored events to client REQ via author public key.
- [X] Store events in SQLite database.
- [X] Register connected clients via Hub-and-Spoke pattern.
- [X] Receive a NIP-01 text event and respond with a NIP-20 OK.

## Usage

```shell
# Build binary
make build

# Run relay
./relay "localhost:8000"
```

From another terminal, send a text event via [melange](https://github.com/ffiat/melange):

```shell
./melange event -note "hello, friend"

[+] Text note published
[
    "OK",
    "4586db2f00bd7a01ec74ee30e514143a7ffbd68eae763ac5c32c07061beede90",
    true
]
```

## Fix

- [X] Connection closes when a client disconnects.
