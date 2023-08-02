package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/net/websocket"
)

func main() {

	origin := "http://localhost/"
	url := "ws://localhost:8000/"

	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	_, err = ws.Write([]byte("hello, friend!\n"))
    if err != nil {
		log.Fatal(err)
	}

    done := make(chan struct{})
	go func() {
		io.Copy(os.Stdout, ws) // NOTE: ignoring errors
		log.Println("done")
		done <- struct{}{} // signal the main goroutine
	}()
	mustCopy(ws, os.Stdin)
	ws.Close()
	<-done // wait for background goroutine to finish

	var msg = make([]byte, 512)
	var n int
	if n, err = ws.Read(msg); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Received: %s.\n", msg[:n])
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}
