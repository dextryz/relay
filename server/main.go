package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

func Env(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("address env variable \"%s\" not set, usual", key)
	}
	return value
}

var RELAY_URL = Env("RELAY_URL")

type client chan<- string // outgoing message channel

var (
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan string) // all incoming messages
)

// 1. The local variable clients records the set of connected clients.
// 2. The only information recorded about each client, is its outgoing channel ID.
// 3. The broadcaster has to listen to the entering and leaving channels for accountments on arriving and departing clients.
// 4. The broadcaster also has to listen to the global message channel for incoming messages.
// 5. When an event is received it has to broadcast this message to each connected client.

func broadcast() {
    clients := make(map[client]bool) // all connected clients

    for {
        // Multiplex client and message channels
        select {
        case m := <-messages:
            // Send event to each client
            for c := range clients {
                c <- m
            }
        case c := <-entering:
            // Set client bool in map
            clients[c] = true
        case c := <-leaving:
            // Close the client channel
            delete(clients, c)
            close(c)
        default:
        }
    }
}

// 1. Create a new receiving message channel for each connecting client.
// 2. Annouce the arrival of this client to the broadcaster over the entering channel.
// 3. Reads every line from the client and send it to the broadcaster over the message channel.
// 4. Prefix each message with the ID of the sender.
// 5. Once there is nothing left too read, EOSE, the handler announces the departure of the client and closes the connecion.

func clientWriter(conn net.Conn, ch <-chan string) {
    for msg := range ch {
        fmt.Fprint(conn, msg)
    }
}

// Echo the data received on the WebSocket.
func handleEvents(conn *websocket.Conn) {

    ch := make(chan string)

    go clientWriter(conn, ch)

    name := ""
    ch <- "Enter your name: "

    input := bufio.NewScanner(conn)
    for input.Scan() {
        if name == "" {
            name = input.Text()
            ch <- "Hi, " + name + "\n"
            messages <- name + " joined" + "\n"
            entering <- ch
        } else {
            messages <- name + ": " + input.Text() + "\n"
        }
    }

    // NOTE: ignoring potential errors from input.Err()

    //leaving <- ch
    //messages <- who + " has left"
    //conn.Close()

    conn.Write([]byte("hello"))
}

// This example demonstrates a trivial echo server.
func main() {

	go broadcast()

	http.Handle("/", websocket.Handler(handleEvents))

	err := http.ListenAndServe(RELAY_URL, nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
