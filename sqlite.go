package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ffiat/nostr"
)

const createDb string = `
DROP TABLE IF EXISTS events;
CREATE TABLE events (
    id TEXT PRIMARY KEY,
    pubkey TEXT,
    created_at INTEGER,
    kind INTEGER,
    content TEXT,
    sig TEXT
);
`

func newSqlite(database string) *sql.DB {

	db, err := sql.Open("sqlite3", database)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(createDb)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("table events created")

    return db
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

func (s *relay) query(filter nostr.Filter) (chan *nostr.Event, error) {

    log.Println("Querying")

    stream := make(chan *nostr.Event, 3)

    for _, pub := range filter.Authors {
        err := eventsByPubkey(s.db, pub, stream)
        if err != nil {
            log.Fatalln(err)
        }
    }

    log.Printf("Len: %d", len(stream))

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

func eventsByPubkey(db *sql.DB, pubkey string, stream chan<- *nostr.Event) error {

    log.Printf("Query by PubKey: %s", pubkey)

    rows, err := db.Query("SELECT id, pubkey, created_at, kind, content, sig FROM events WHERE pubkey = ?", pubkey)
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var event nostr.Event
        err := rows.Scan(&event.Id, &event.PubKey, &event.CreatedAt, &event.Kind, &event.Content, &event.Sig)
        if err != nil {
            return err
        }
        stream <- &event
    }

    err = rows.Err()
    if err != nil {
        return err
    }

    log.Print("--------")

    return nil
}
