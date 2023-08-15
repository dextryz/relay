package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"github.com/ffiat/nostr"
)

func main() {

	listener, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("listening on http://%v", listener.Addr())

	db := newSqlite("test.db")
	defer db.Close()

	info := nostr.RelayInformation{
		Name:          "ffiat",
		Description:   "Tiny Relay",
		PubKey:        "",
		Contact:       "",
		SupportedNIPs: []int{1, 11, 16, 20},
		Version:       "n/a",
		Limitation: &nostr.RelayLimitation{
			MaxLimit:   500,
			MaxFilters: 500,
		},
	}

	relay := newRelay(db, info)

	go relay.broadcaster()

	s := &http.Server{
		Handler: relay,
	}

	err = s.Serve(listener)
	if err != nil {
		log.Fatalln(err)
	}
}
