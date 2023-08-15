module github.com/ffiat/relay

go 1.19

replace github.com/ffiat/nostr => ../nostr

require (
	github.com/ffiat/nostr v0.1.1
	github.com/gobwas/ws v1.0.2
	github.com/mattn/go-sqlite3 v1.14.17
)

require (
	github.com/btcsuite/btcd/btcec/v2 v2.3.2 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.3 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.1 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.0.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/gobwas/httphead v0.0.0-20180130184737-2c6c146eadee // indirect
	github.com/gobwas/pool v0.2.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
)
