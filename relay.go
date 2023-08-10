package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"log"
	"net/http"

	"nhooyr.io/websocket"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ffiat/nostr"
)

type writer struct {
    connection *websocket.Conn
}

// 1. Relay managed the database.

type relay struct {
	db         *sql.DB
    ws    *websocket.Conn
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

	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Fatalln(err)
		return
	}
	//defer ws.Close(websocket.StatusInternalError, "closing")

    s.ws = ws

	c := client{
		send:   make(chan nostr.Event),
		result: make(chan nostr.MessageOk),
	}

	s.register <- c

	ctx := r.Context()
	//ctx := ws.CloseRead(r.Context())

    for {

        kind, raw, err := ws.Read(ctx)
        if err != nil {
            log.Fatalln(err)
        }

        msg, err := s.process(raw)
        if err != nil {
            log.Fatalln(err)
        }

        err = ws.Write(ctx, kind, msg)
        if err != nil {
            log.Fatalln(err)
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

	log.Printf("Event (id: %s) stored in relay DB", e.Id)

	return nil
}

func nextMessage(ctx context.Context, ws *websocket.Conn) ([]byte, error) {

	typ, b, err := ws.Read(ctx)
	if err != nil {
		return nil, err
	}

	if typ != websocket.MessageText {
		ws.Close(websocket.StatusUnsupportedData, "expected text message")
		return nil, fmt.Errorf("expected text message but got %v", typ)
	}
	return b, nil
}
