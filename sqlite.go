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

	log.Printf("Event (id: %s, pubkey: [%s..%s]) stored in relay DB", e.Id[:10], e.PubKey[:5], e.PubKey[len(e.PubKey)-5:])

	return nil
}

// Query database with a filter for a set of events, then push (SubId, Event) on the stream.
func (s *relay) query(subId string, filter nostr.Filter, stream chan<- nostr.MessageEvent) error {

	if len(filter.Authors) > s.limits.max_limit {
		return ErrMaxLimit
	}

	if len(filter.Ids) > s.limits.max_limit {
		return ErrMaxLimit
	}

	for _, pub := range filter.Authors {

		events, err := eventsByPubkey(s.db, pub)
		if err != nil {
			return err
		}

		for _, event := range events {
			stream <- nostr.MessageEvent{
				SubscriptionId: subId,
				Event:          *event,
			}
		}

		log.Printf("Pulled events with public key: [%s..%s]", pub[:5], pub[len(pub)-5:])
	}

	for _, id := range filter.Ids {

		event, err := eventById(s.db, id)
		if err != nil {
			return err
		}

		stream <- nostr.MessageEvent{
			SubscriptionId: subId,
			Event:          *event,
		}

		log.Printf("Pulled event with ID: [%s..%s]", id[:5], id[len(id)-5:])
	}

	return nil
}

func eventById(db *sql.DB, id string) (*nostr.Event, error) {

	event := &nostr.Event{}

	row := db.QueryRow("SELECT id, pubkey, created_at, kind, content, sig FROM events WHERE id = ?", id)

	err := row.Scan(&event.Id, &event.PubKey, &event.CreatedAt, &event.Kind, &event.Content, &event.Sig)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func eventsByPubkey(db *sql.DB, pubkey string) ([]*nostr.Event, error) {

	var events []*nostr.Event

	rows, err := db.Query("SELECT id, pubkey, created_at, kind, content, sig FROM events WHERE pubkey = ?", pubkey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var event nostr.Event
		err := rows.Scan(&event.Id, &event.PubKey, &event.CreatedAt, &event.Kind, &event.Content, &event.Sig)
		if err != nil {
			return nil, err
		}
		events = append(events, &event)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return events, nil
}
