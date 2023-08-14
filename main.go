package main

import (
	"log"
	"net"
	"net/http"
	"os"
)

func main() {

	listener, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("listening on http://%v", listener.Addr())

	db := newSqlite("test.db")
	defer db.Close()

	relay := newRelay(db)

	go relay.broadcaster()

	s := &http.Server{
		Handler: relay,
	}

	err = s.Serve(listener)
	if err != nil {
		log.Fatalln(err)
	}

}
