// Package events appends to the event log (E1). Rows are never updated or deleted, except
// when an import replaces the whole log (X1).
package events

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Event kinds. Payloads are JSON, so adding a kind needs no migration. Older logs may also
// hold backlog_snapshot events, no longer written or read (ADR 0016).
const (
	StatusChanged    = "status_changed"    // {"from","to","source"}
	Rated            = "rated"             // {"rating"}
	PlaytimeSnapshot = "playtime_snapshot" // {"games","new"} once per sync
	RecShown         = "rec_shown"         // {"row"}: the section it was shown in
	RecOpened        = "rec_opened"
	RecDismissed     = "rec_dismissed"
)

// Execer is satisfied by *sql.DB and *sql.Tx so events join the caller's transaction.
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// Log appends one event. appid 0 means the event is not about a game.
func Log(db Execer, at time.Time, kind string, appid int, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var id any
	if appid != 0 {
		id = appid
	}
	_, err = db.Exec(`INSERT INTO event (at, kind, appid, payload) VALUES (?, ?, ?, ?)`,
		at.UTC().Format(time.RFC3339), kind, id, string(b))
	return err
}
