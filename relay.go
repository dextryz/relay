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
        log.Println(string(msg))
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

    log.Println("process")
    log.Println(msg)

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
		r := nostr.MessageOk{
			EventId: e.GetId(),
			Ok:      true,
			Message: "",
		}

		return json.Marshal(r)

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

    log.Printf("Event (id: %s, pubkey: %s) stored in relay DB", e.Id[:16], e.PubKey)

	return nil
}

func (s *relay) query(filter nostr.Filter) (<-chan *nostr.Event, error) {

    log.Println("Querying")

    stream := make(chan *nostr.Event)

    for _, pub := range filter.Authors {
        events, err := eventsByPubkey(s.db, pub)
        if err != nil {
            return nil, err
        }
        for _, e := range events {
            stream <- &e
        }
    }

//     for _, id := range filter.Ids {
//         e, err := getEvent(s.db, id)
//         if err != nil {
//             return nil, err
//         }
//         stream <- e
//     }

    return stream, nil
}

func getEvent(db *sql.DB, id string) (*nostr.Event, error) {

    event := &nostr.Event{}

    row := db.QueryRow("SELECT id, pubkey, created_at, kind, content, sig FROM events WHERE id = ?", id)

    err := row.Scan(&event.Id, &event.PubKey, &event.CreatedAt, &event.Kind, &event.Content, &event.Sig)
    if err != nil {
        return nil, err
    }

    return event, nil
}

func eventsByPubkey(db *sql.DB, pubkey string) ([]nostr.Event, error) {

    log.Printf("Query by PubKey: %s", pubkey)

    rows, err := db.Query("SELECT id, pubkey, created_at, kind, content, sig FROM events WHERE pubkey = ?", pubkey)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var events []nostr.Event
    for rows.Next() {
        var event nostr.Event
        if err := rows.Scan(&event.Id, &event.PubKey, &event.CreatedAt, &event.Kind, &event.Content, &event.Sig); err != nil {
            return nil, err
        }
        log.Println("EVENT FOUND")
        log.Println(event)
        events = append(events, event)
    }

    if err := rows.Err(); err != nil {
        return nil, err
    }

    return events, nil
}
