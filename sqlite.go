package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

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

	var conditions []string
	var args []interface{}

	if len(filter.Ids) > 0 {

		if len(filter.Ids) > s.info.Limitation.MaxLimit {
			return nil
		}

		placeholdersForIds := make([]string, len(filter.Ids))
		for i, id := range filter.Ids {
			placeholdersForIds[i] = "?"
			args = append(args, id)
		}
		conditions = append(conditions, fmt.Sprintf("id IN (%s)", strings.Join(placeholdersForIds, ",")))
	}

	if len(filter.Authors) > 0 {

		if len(filter.Authors) > s.info.Limitation.MaxLimit {
			return nil
		}

		placeholdersForPubkeys := make([]string, len(filter.Authors))
		for i, pubkey := range filter.Authors {
			placeholdersForPubkeys[i] = "?"
			args = append(args, pubkey)
		}
		conditions = append(conditions, fmt.Sprintf("pubkey IN (%s)", strings.Join(placeholdersForPubkeys, ",")))
	}

	if filter.Kinds != nil {

		if len(filter.Kinds) > 10 {
			// too many kinds, fail everything
			return nil
		}

		if len(filter.Kinds) == 0 {
			// kinds being [] mean you won't get anything
			return nil
		}

		// no sql injection issues since these are ints
		inkinds := make([]string, len(filter.Kinds))
		for i, kind := range filter.Kinds {
			inkinds[i] = strconv.Itoa(int(kind))
		}
		conditions = append(conditions, fmt.Sprintf("kind IN (%s)", strings.Join(inkinds, ",")))
	}

	// If both lists are empty, this is an error or not meaningful.
	if len(conditions) == 0 {
		return fmt.Errorf("both ids and pubkeys are empty")
	}

	query := fmt.Sprintf("SELECT * FROM events WHERE %s", strings.Join(conditions, " AND "))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	var events []*nostr.Event

	for rows.Next() {
		var event nostr.Event
		err := rows.Scan(&event.Id, &event.PubKey, &event.CreatedAt, &event.Kind, &event.Content, &event.Sig)
		if err != nil {
			return err
		}
		events = append(events, &event)
	}

	for _, event := range events {
		stream <- nostr.MessageEvent{
			SubscriptionId: subId,
			Event:          *event,
		}
	}

	return nil
}
