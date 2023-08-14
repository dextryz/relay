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

    // TODO: Maybe add client ID and Authentication via NIP-42
    log.Println("client connected")

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

            log.Println("SENDING event to clienr")

            bytes, err := json.Marshal(msg)
            if err != nil {
                log.Fatalln("unable to send REQ filtered messages")
            }

            err = wsutil.WriteServerMessage(conn, ws.OpText, bytes)
            if err != nil {
                log.Printf("Err 3: %#v", err)
            }
        case msg := <-c.result:

            log.Println("SENDING OK response to clienr")

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
		msg, _, err := wsutil.ReadClientData(conn)
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

		msg, err = s.process(&c, msg)
		if err != nil {
			log.Printf("Err 2: %#v", err)
		}
	}
}

func (s relay) process(c *client, raw []byte) ([]byte, error) {

	msg := nostr.DecodeMessage(raw)

	switch msg.Type() {
	case "EVENT":

		log.Print("EVENT")

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
		c.result <- nostr.MessageOk{
			EventId: e.GetId(),
			Ok:      true,
			Message: "",
		}

	case "REQ":
		log.Print("REQ")

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
        for _, filter := range msg.Filters {

            eventStream, err := s.query(filter)
            if err != nil {
                log.Fatalf("unable to query events: %v", err)
            }

            if len(eventStream) == 0 {
                log.Println("no events found")
            }

            for event := range eventStream {

                log.Printf("E: %v", event)

                // Needc to put ths here to get SubId
                m := nostr.MessageEvent{
                    SubscriptionId: msg.SubscriptionId,
                    Event: *event,
                }

                c.send <- m
            }
        }

        // 3. Send these events to the current spoke's send channel.
        // There is no need to broadcast it to the hub, since we want to send the data to the current client.
        // We are basically just making a round trip to the event repository.

		return nil, nil
    default:
        log.Fatalln("unkown event type")
	}

	return nil, nil
}

func (s *relay) broadcaster() {

	for {
		select {
		case e := <-s.events:
			for c := range s.clients {

                // FIXME: get SubId
                m := nostr.MessageEvent{
                    SubscriptionId: "",
                    Event: e,
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
