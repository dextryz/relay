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
: go build

# Run relay
: ./relay "localhost:8000"
```

From another terminal, send a text event via [melange](https://github.com/ffiat/melange):

```shell
: ./melange event -note "hello, friend"

[+] Text note published
[
    "OK",
    "4586db2f00bd7a01ec74ee30e514143a7ffbd68eae763ac5c32c07061beede90",
    true
]
```
