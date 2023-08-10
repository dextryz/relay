package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ffiat/nostr"
)

type relay struct {
	db         *sql.DB
	clients    map[*client]bool
	events     chan nostr.Event
	register   chan client
	unregister chan client
}

func newRelay(db *sql.DB) *relay {
	return &relay{
		db:         db,
		clients:    make(map[*client]bool),
		events:     make(chan nostr.Event),
		register:   make(chan client),
		unregister: make(chan client),
	}
}

func (s *relay) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ws, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Fatalln(err)
		return
	}
	defer ws.Close()

	c := client{
		send:   make(chan nostr.Event),
		result: make(chan nostr.MessageOk),
	}

	s.register <- c

	for {
		msg, op, err := wsutil.ReadClientData(ws)
		if err != nil {
			if strings.Contains(err.Error(), "ws closed: 1000") {
				log.Println("client disconnected")
				break
			}
			if err != io.EOF {
				log.Printf("Read error: %v", err)
			}
			// The client closed the connection.
			// So break out of the loop and end the goroutine witht the close statement.
			break
		}

		msg, err = s.process(msg)
		if err != nil {
			log.Printf("Err 2: %#v", err)
		}

		err = wsutil.WriteServerMessage(ws, op, msg)
		if err != nil {
			log.Printf("Err 3: %#v", err)
		}
	}
}

func (s relay) process(raw []byte) ([]byte, error) {

	msg := nostr.DecodeMessage(raw)
	switch msg.Type() {
	case "EVENT":

		var e nostr.MessageEvent
		err := json.Unmarshal(raw, &e)
		if err != nil {
			log.Fatalf("unable to unmarshal event: %v", err)
		}

		err = s.store(e.Event)
		if err != nil {
			log.Fatalf("unable to store event: %v", err)
		}

		// Return the result as defined in NIP-20
		r := nostr.MessageOk{
			EventId: e.GetId(),
			Ok:      true,
			Message: "",
		}

		return json.Marshal(r)

	case "REQ":
		log.Print("REQ")
		return nil, nil
	}

	return nil, nil
}

func (s *relay) broadcaster() {

	for {
		select {
		case e := <-s.events:
			for c := range s.clients {
				c.send <- e
			}
		case c := <-s.register:
			s.clients[&c] = true
		case c := <-s.unregister:
			delete(s.clients, &c)
			c.Close()
		default:
		}
	}
}

func (s *relay) store(e nostr.Event) error {

	eventSql := "INSERT INTO events (id, pubkey, created_at, kind, content, sig) VALUES (?, ?, ?, ?, ?, ?)"

	_, err := s.db.Exec(eventSql, e.Id, e.PubKey, e.CreatedAt, e.Kind, e.Content, e.Sig)
	if err != nil {
		return err
	}

	log.Printf("Event (id: %s) stored in relay DB", e.Id[:16])

	return nil
}
