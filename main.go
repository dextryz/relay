package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
)

const createDb string = `
DROP TABLE IF EXISTS events;
CREATE TABLE events (
    id TEXT PRIMARY KEY,
    pubkey TEXT,
    created_at INTEGER,
    kind INTEGER,
    content TEXT,
    sig TEXT
);
`

func main() {

	listener, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("listening on http://%v", listener.Addr())

	db, err := sql.Open("sqlite3", "test.db")
	if err != nil {
		log.Println("elfweklfklj")
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(createDb)
	if err != nil {
		log.Println("elfweklfklj")
		log.Fatal(err)
	}

	log.Println("table events created")

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
