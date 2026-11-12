// Package transfer handles the database backup download (B1) and the JSON export and
// import (X1). Neither ever contains the API key; store data is re-fetched, not exported.
package transfer

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"somanygames/internal/library"
	"somanygames/internal/store"
)

// FormatVersion is bumped when the export layout changes. Version 2 stores ratings as
// tier letters (ADR 0020); version 1 files, with 0-10 scores, are converted on import.
const FormatVersion = 2

// File is the export format.
type File struct {
	Version    int         `json:"version"`
	ExportedAt string      `json:"exported_at"`
	Games      []GameState `json:"games"`
	Tags       TagSettings `json:"tag_settings"`
	Events     []Event     `json:"events"`
	// Tiers is the tier list (T2); absent in files from before it existed, which leave it as is.
	Tiers []library.TierEntry `json:"tiers"`
}

// GameState is the owner's data about one game.
type GameState struct {
	AppID           int     `json:"appid"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	Completed       bool    `json:"completed"`
	Rating          *string `json:"rating,omitempty"`
	FinishedAt      *string `json:"finished_at,omitempty"`
	HideOverride    string  `json:"hide_override,omitempty"`
	Classified      bool    `json:"classified"`
	MetaUnavailable bool    `json:"meta_unavailable,omitempty"`
	FirstSeenAt     string  `json:"first_seen_at"`
}

// TagSettings are the tag-related settings carried over (X1).
type TagSettings struct {
	Stoplist       []string `json:"stoplist"`
	Pinned         []string `json:"pinned"`
	Avoided        []string `json:"avoided"`
	HiddenPatterns []string `json:"hidden_patterns"`
}

// Event is one event-log row.
type Event struct {
	At      string          `json:"at"`
	Kind    string          `json:"kind"`
	AppID   *int            `json:"appid,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

// Backup writes a copy of the database to dest (B1).
func Backup(db *sql.DB, dest string) error { return store.Snapshot(db, dest) }

// Export reads everything X1 carries.
func Export(db *sql.DB, now time.Time) (File, error) {
	f := File{Version: FormatVersion, ExportedAt: now.UTC().Format(store.TimeFormat)}
	rows, err := db.Query(`SELECT appid, name, status, completed, rating, finished_at, hide_override, classified, meta_unavailable, first_seen_at
		FROM game ORDER BY appid`)
	if err != nil {
		return f, err
	}
	for rows.Next() {
		var g GameState
		var rating, finished sql.NullString
		if err := rows.Scan(&g.AppID, &g.Name, &g.Status, &g.Completed, &rating, &finished, &g.HideOverride, &g.Classified, &g.MetaUnavailable, &g.FirstSeenAt); err != nil {
			rows.Close()
			return f, err
		}
		if rating.Valid {
			g.Rating = &rating.String
		}
		if finished.Valid {
			g.FinishedAt = &finished.String
		}
		f.Games = append(f.Games, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return f, err
	}

	s, err := store.LoadSettings(db)
	if err != nil {
		return f, err
	}
	f.Tags = TagSettings{s.Stoplist, s.Pinned, s.Avoided, s.HiddenPatterns}
	if f.Tiers, err = library.LoadTiers(db); err != nil {
		return f, err
	}

	evs, err := db.Query(`SELECT at, kind, appid, payload FROM event ORDER BY id`)
	if err != nil {
		return f, err
	}
	defer evs.Close()
	for evs.Next() {
		var e Event
		var appid sql.NullInt64
		var payload string
		if err := evs.Scan(&e.At, &e.Kind, &appid, &payload); err != nil {
			return f, err
		}
		if appid.Valid {
			id := int(appid.Int64)
			e.AppID = &id
		}
		e.Payload = json.RawMessage(payload)
		f.Events = append(f.Events, e)
	}
	return f, evs.Err()
}

// Parse reads and validates an export file.
func Parse(data []byte) (File, error) {
	var f File
	var head struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return f, fmt.Errorf("not a So Many Games export: %w", err)
	}
	if head.Version == 1 {
		var err error
		if data, err = upgradeV1(data); err != nil {
			return f, fmt.Errorf("not a So Many Games export: %w", err)
		}
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return f, fmt.Errorf("not a So Many Games export: %w", err)
	}
	if f.Version != FormatVersion {
		return f, fmt.Errorf("unsupported export version %d", f.Version)
	}
	for _, g := range f.Games {
		if g.FinishedAt != nil {
			if _, err := time.Parse(library.DateFormat, *g.FinishedAt); err != nil {
				return f, fmt.Errorf("game %d: bad finished date", g.AppID)
			}
		}
		if _, err := library.ParseStatus(g.Status); err != nil {
			return f, fmt.Errorf("game %d: %w", g.AppID, err)
		}
		if g.Rating != nil && !library.ValidRating(*g.Rating) {
			return f, fmt.Errorf("game %d: rating must be S, A, B, C, D or F", g.AppID)
		}
	}
	return f, nil
}

// Summary describes what an import would change, shown before asking (X1).
type Summary struct {
	GamesInFile     int  `json:"gamesInFile"`
	NewGames        int  `json:"newGames"`
	StatusChanges   int  `json:"statusChanges"`
	RatingChanges   int  `json:"ratingChanges"`
	HiddenChanges   int  `json:"hiddenChanges"`
	EventsInFile    int  `json:"eventsInFile"`
	EventsNow       int  `json:"eventsNow"`
	TagSettingsDiff bool `json:"tagSettingsDiff"`
}

// Summarise compares f with the current database.
func Summarise(db *sql.DB, f File) (Summary, error) {
	sum := Summary{GamesInFile: len(f.Games), EventsInFile: len(f.Events)}
	current, err := Export(db, time.Now())
	if err != nil {
		return sum, err
	}
	sum.EventsNow = len(current.Events)
	byID := map[int]GameState{}
	for _, g := range current.Games {
		byID[g.AppID] = g
	}
	for _, g := range f.Games {
		cur, ok := byID[g.AppID]
		if !ok {
			sum.NewGames++
			continue
		}
		if cur.Status != g.Status || cur.Completed != g.Completed || cur.Classified != g.Classified {
			sum.StatusChanges++
		}
		if !sameRating(cur.Rating, g.Rating) {
			sum.RatingChanges++
		}
		if cur.HideOverride != g.HideOverride {
			sum.HiddenChanges++
		}
	}
	a, _ := json.Marshal(current.Tags)
	b, _ := json.Marshal(f.Tags)
	sum.TagSettingsDiff = string(a) != string(b)
	return sum, nil
}

// Apply replaces the owner's data with f: game fields, tag settings and the event log.
// Games in the library but not in the file keep their data.
func Apply(db *sql.DB, f File, now time.Time) error {
	if f.Version != FormatVersion {
		return errors.New("unsupported export version")
	}
	// Read settings before the transaction: the pool has a single connection.
	s, err := store.LoadSettings(db)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, g := range f.Games {
		first := g.FirstSeenAt
		if first == "" {
			first = now.UTC().Format(store.TimeFormat)
		}
		if _, err := tx.Exec(`INSERT INTO game (appid, name, status, completed, rating, finished_at, hide_override, classified, meta_unavailable, first_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (appid) DO UPDATE SET status = excluded.status, completed = excluded.completed,
				rating = excluded.rating, finished_at = excluded.finished_at, hide_override = excluded.hide_override,
				classified = excluded.classified, meta_unavailable = excluded.meta_unavailable`,
			g.AppID, g.Name, g.Status, g.Completed, g.Rating, g.FinishedAt, g.HideOverride, g.Classified, g.MetaUnavailable, first); err != nil {
			return err
		}
	}
	if f.Tiers != nil {
		if err := library.WriteTiers(tx, f.Tiers); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM event`); err != nil {
		return err
	}
	for _, e := range f.Events {
		payload := string(e.Payload)
		if payload == "" {
			payload = "{}"
		}
		if _, err := tx.Exec(`INSERT INTO event (at, kind, appid, payload) VALUES (?, ?, ?, ?)`, e.At, e.Kind, e.AppID, payload); err != nil {
			return err
		}
	}
	s.Stoplist, s.Pinned, s.Avoided, s.HiddenPatterns = f.Tags.Stoplist, f.Tags.Pinned, f.Tags.Avoided, f.Tags.HiddenPatterns
	if err := library.RecomputeHidden(tx, s.HiddenPatterns); err != nil {
		return err
	}
	if err := store.WriteSettings(tx, s); err != nil {
		return err
	}
	return tx.Commit()
}

// upgradeV1 rewrites a version 1 export as version 2: 0-10 ratings on games and in
// 'rated' events become tier letters (ADR 0020).
func upgradeV1(data []byte) ([]byte, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	letter := func(obj map[string]json.RawMessage) error {
		raw, ok := obj["rating"]
		if !ok || string(raw) == "null" {
			return nil
		}
		var n int
		if err := json.Unmarshal(raw, &n); err != nil {
			return fmt.Errorf("rating %s is not a 0-10 score", raw)
		}
		if n < 0 || n > 10 {
			return fmt.Errorf("rating %d out of range", n)
		}
		obj["rating"], _ = json.Marshal(library.RatingFromScore(n))
		return nil
	}
	if raw, ok := doc["games"]; ok && string(raw) != "null" {
		var games []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &games); err != nil {
			return nil, err
		}
		for _, g := range games {
			if err := letter(g); err != nil {
				return nil, err
			}
		}
		doc["games"], _ = json.Marshal(games)
	}
	if raw, ok := doc["events"]; ok && string(raw) != "null" {
		var evs []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &evs); err != nil {
			return nil, err
		}
		for _, e := range evs {
			var kind string
			json.Unmarshal(e["kind"], &kind)
			if kind != "rated" {
				continue
			}
			var payload map[string]json.RawMessage
			if json.Unmarshal(e["payload"], &payload) != nil {
				continue
			}
			if err := letter(payload); err != nil {
				return nil, err
			}
			e["payload"], _ = json.Marshal(payload)
		}
		doc["events"], _ = json.Marshal(evs)
	}
	doc["version"], _ = json.Marshal(FormatVersion)
	return json.Marshal(doc)
}

func sameRating(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
