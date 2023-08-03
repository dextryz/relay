package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"nhooyr.io/websocket"

	"github.com/ffiat/nostr"
)

type relay struct {
	clients    map[*client]bool
	events     chan nostr.Event
	register   chan client
	unregister chan client
}

func newRelay() *relay {

	r := relay{
		clients:    make(map[*client]bool),
		events:     make(chan nostr.Event),
		register:   make(chan client),
		unregister: make(chan client),
	}

	return &r
}

func (s *relay) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Fatalln(err)
		return
	}
	defer ws.Close(websocket.StatusInternalError, "")

	c := client{
		send:   make(chan nostr.Event),
		result: make(chan nostr.MessageOk),
	}

	s.register <- c

	ctx := context.Background()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-c.send:

				bytes, err := json.Marshal(e)
				if err != nil {
					log.Fatal(err)
					return
				}

				err = ws.Write(context.TODO(), websocket.MessageText, bytes)
				if err != nil {
					log.Fatal(err)
					return
				}
			case r := <-c.result:

				bytes, err := json.Marshal(r)
				if err != nil {
					log.Fatal(err)
					return
				}

				err = ws.Write(context.TODO(), websocket.MessageText, bytes)
				if err != nil {
					log.Fatal(err)
					return
				}
			}
		}
	}()

	// Pull events from broker into local channels.
	// Defining a function creates a smaller lexical scope for confinement.
	readStream := func() <-chan []byte {
		wg.Add(1)
		stream := make(chan []byte)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, raw, err := ws.Read(ctx)
					if err != nil {
						log.Fatalln(err)
					}

					stream <- raw
				}
			}
		}()
		return stream
	}

	for raw := range readStream() {

		msg := nostr.DecodeMessage(raw)

		switch msg.Type() {
		case "EVENT":

			var e nostr.MessageEvent
			err = json.Unmarshal(raw, &e)
			if err != nil {
				log.Fatalf("unable to unmarshal event: %v", err)
			}

			err := s.store(e.Event)
			if err != nil {
				log.Fatalf("unable to store event: %v", err)
			}

            // Return the result as defined in NIP-20
			c.result <- nostr.MessageOk{
				EventId: e.GetId(),
				Ok:      true,
				Message: "",
			}

            // TODO: Broadcast events to registered clients.
			//s.events <- e.Event

		case "REQ":
			log.Print("REQ")
		}
	}

	wg.Wait()
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

	// TODO: Store in relay database

	return nil
}
