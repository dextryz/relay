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
	db      *sql.DB
	info    nostr.RelayInformation
	clients map[*client]bool

	events     chan nostr.Event
	register   chan client
	unregister chan client
}

func newRelay(db *sql.DB, info nostr.RelayInformation) *relay {
	return &relay{
		db:         db,
		info:       info,
		clients:    make(map[*client]bool),
		events:     make(chan nostr.Event),
		register:   make(chan client),
		unregister: make(chan client),
	}
}

func (s *relay) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// TODO: Maybe add client ID and Authentication via NIP-42
	log.Println("client connected")

	// Support for NIP-11 - Send relay information to client.
	if r.Header.Get("Accept") == "application/nostr+json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.info)
	} else {
		s.HandleWebsocket(w, r)
	}
}

func (s *relay) HandleWebsocket(w http.ResponseWriter, r *http.Request) {

	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Fatalln(err)
		return
	}
	defer conn.Close()

	c := client{
		send:   make(chan nostr.MessageEvent),
		result: make(chan nostr.MessageOk),
	}

	s.register <- c

	go func() {
		select {
		case msg := <-c.send:

			log.Println("sending EVENT to client")

			bytes, err := json.Marshal(msg)
			if err != nil {
				log.Fatalln("unable to send REQ filtered messages")
			}

			err = wsutil.WriteServerMessage(conn, ws.OpText, bytes)
			if err != nil {
				log.Printf("Err 3: %#v", err)
			}
		case msg := <-c.result:

			log.Println("sending OK to client")

			bytes, err := json.Marshal(msg)
			if err != nil {
				log.Fatalln("unable to send Ok response")
			}

			err = wsutil.WriteServerMessage(conn, ws.OpText, bytes)
			if err != nil {
				log.Printf("Err 3: %#v", err)
			}
		}
	}()

	for {
		raw, _, err := wsutil.ReadClientData(conn)
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

		// Process message depending on the type.
		msg := nostr.DecodeMessage(raw)
		switch msg.Type() {
		case "EVENT":
			res, err := s.storeEvent(raw)
			if err != nil {
				log.Fatalln(err)
			}
			c.result <- res
		case "REQ":
			// Pull events into a read-only channel.
			events, err := s.pullEvents(raw)
			if err != nil {
				log.Fatalln(err)
			}
			for e := range events {
				c.send <- e
			}
		default:
			log.Fatalln("unkown event type")
		}
	}
}

// Store event in database and return result command (NIP-20)
func (s *relay) storeEvent(raw []byte) (nostr.MessageOk, error) {

	var e nostr.MessageEvent
	err := json.Unmarshal(raw, &e)
	if err != nil {
		log.Fatalf("unable to unmarshal event: %v", err)
	}

    if e.IsBasicEvent() || e.IsRegularEvent() {
        err = s.store(e.Event)
        if err != nil {
            log.Fatalf("unable to store event: %v", err)
        }
    }

    if e.IsReplaceableEvent() {
        log.Fatalf("replaceable event types [%d] to supported yet", e.Event.Kind)
    }

    if e.IsEphemeralEvent() {
        log.Fatalln("ephemeral event types to supported yet.")
    }

	// Return the result as defined in NIP-20
	return nostr.MessageOk{
		EventId: e.GetId(),
		Ok:      true,
		Message: "",
	}, nil
}

// Use a confined stream to pull REQ events from database.
// Note that we have to encode the message subID in this method
func (s *relay) pullEvents(raw []byte) (<-chan nostr.MessageEvent, error) {

	// 1. Parse the req message from the raw stream of data.
	var msg nostr.MessageReq
	err := json.Unmarshal(raw, &msg)
	if err != nil {
		log.Fatalf("unable to unmarshal event: %v", err)
	}

	if len(msg.Filters) == 0 {
		log.Println("no filters to be applied")
	}

	// 2. Query the event repository with the filter and get a set of events.
	stream := make(chan nostr.MessageEvent)

	go func() {
		defer close(stream)

		for _, filter := range msg.Filters {

			err := s.query(msg.SubscriptionId, filter, stream)
			if err != nil {
				log.Fatalf("unable to query events: %v", err)
			}

			for event := range stream {
				stream <- event
			}
		}
	}()

	// 3. Send these events to the current spoke's send channel.
	// There is no need to broadcast it to the hub, since we want to send the data to the current client.
	// We are basically just making a round trip to the event repository.

	return stream, nil
}

func (s *relay) broadcaster() {

	for {
		select {
		case e := <-s.events:
			for c := range s.clients {

				// FIXME: get SubId
				m := nostr.MessageEvent{
					SubscriptionId: "",
					Event:          e,
				}

				c.send <- m
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
