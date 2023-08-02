package main

import (
	"flag"
	"io"
	"log"
	"os"

	"golang.org/x/net/websocket"
)

var name = flag.String("name", "", "client user name")

func Env(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("address env variable \"%s\" not set, usual", key)
	}
	return value
}

var RELAY_URL = Env("RELAY_URL")

func main() {

    flag.Parse()

    ws, err := websocket.Dial("ws://" + RELAY_URL, "", "http://")
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
}

func mustCopy(dst io.Writer, src io.Reader) {
	_, err := io.Copy(dst, src)
    if err != nil {
		log.Fatal(err)
	}
}
