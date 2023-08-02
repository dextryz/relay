package main

import (
	"log"
	"net"
	"bufio"
	"fmt"
)

// 1. The local variable clients records the set of connected clients.
// 2. The only information recorded about each client, is its outgoing channel ID.
// 3. The broadcaster has to listen to the entering and leaving channels for accountments on arriving and departing clients.
// 4. The broadcaster also has to listen to the global message channel for incoming messages.
// 5. When an event is received it has to broadcast this message to each connected client.

type client chan<- string // outgoing message channel

var (
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan string) // all incoming messages
)

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

func handleConn(conn net.Conn) {

    ch := make(chan string)

    go clientWriter(conn, ch)

    who := conn.RemoteAddr().String()
    ch <- "You are " + who
    messages <- who + " has arrived"
    entering <- ch

    input := bufio.NewScanner(conn)
    for input.Scan() {
        messages <- who + ": " + input.Text()
    }

    // NOTE: ignoring potential errors from input.Err()

    leaving <- ch
    messages <- who + " has left"
    conn.Close()
}

func clientWriter(conn net.Conn, ch <-chan string) {
    for msg := range ch {
        fmt.Fprintln(conn, msg)
    }
}

func main() {

	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	go broadcast()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go handleConn(conn)
	}
}
