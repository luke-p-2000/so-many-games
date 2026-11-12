package library

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"somanygames/internal/events"
	"somanygames/internal/steam"
	"somanygames/internal/store"
)

// OwnedSource provides the owned-games list (the Steam Web API in the app, a fake in tests).
type OwnedSource interface {
	OwnedGames(ctx context.Context, steamID string) ([]steam.Owned, error)
}

// Service applies library changes. Every status change writes an event.
type Service struct {
	DB     *sql.DB
	Source OwnedSource
	Now    func() time.Time
}

// SyncResult summarises one sync.
type SyncResult struct {
	FirstImport bool `json:"firstImport"`
	NewGames    int  `json:"newGames"`
	AutoChanged int  `json:"autoChanged"`
}

// ErrNotConnected means setup (S1) has not saved a SteamID yet.
var ErrNotConnected = errors.New("Steam is not connected yet")

type existing struct {
	status     Status
	classified bool
	minutes    int
	lastPlayed string
}

// Sync fetches the owned-games list and imports it: new games are added (Unclassified on
// the first import, L7/L8), a snapshot is stored when playtime changes, and L2 runs.
func (s *Service) Sync(ctx context.Context) (SyncResult, error) {
	settings, err := store.LoadSettings(s.DB)
	if err != nil {
		return SyncResult{}, err
	}
	if settings.SteamID == "" {
		return SyncResult{}, ErrNotConnected
	}
	owned, err := s.Source.OwnedGames(ctx, settings.SteamID)
	if err != nil {
		return SyncResult{}, err
	}
	return s.Import(owned, settings)
}

// Import applies an owned-games list; split from Sync so it is testable without a provider.
func (s *Service) Import(owned []steam.Owned, settings store.Settings) (SyncResult, error) {
	now := s.Now()
	stamp := now.UTC().Format(store.TimeFormat)
	res := SyncResult{FirstImport: settings.FirstSyncAt == ""}

	tx, err := s.DB.Begin()
	if err != nil {
		return res, err
	}
	defer tx.Rollback()

	known, err := loadExisting(tx)
	if err != nil {
		return res, err
	}
	for _, o := range owned {
		last := ""
		if o.LastPlayed != nil {
			last = o.LastPlayed.UTC().Format(store.TimeFormat)
		}
		prev, ok := known[o.AppID]
		if !ok {
			res.NewGames++
			if err := s.addGame(tx, o, now, stamp, res.FirstImport, settings.ClassifyDays); err != nil {
				return res, err
			}
		} else {
			if _, err := tx.Exec(`UPDATE game SET name = ? WHERE appid = ?`, o.Name, o.AppID); err != nil {
				return res, err
			}
			if prev.classified {
				next := AfterSync(prev.status, prev.minutes, o.Minutes)
				if next != prev.status {
					res.AutoChanged++
					if err := changeStatus(tx, now, o.AppID, prev.status, next, "sync"); err != nil {
						return res, err
					}
				}
			}
			if prev.minutes == o.Minutes && prev.lastPlayed == last {
				continue
			}
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO playtime_snapshot (appid, taken_at, minutes_total, last_played_at)
			VALUES (?, ?, ?, NULLIF(?, ''))`, o.AppID, stamp, o.Minutes, last); err != nil {
			return res, err
		}
	}
	if err := RecomputeHidden(tx, settings.HiddenPatterns); err != nil {
		return res, err
	}
	if err := events.Log(tx, now, events.PlaytimeSnapshot, 0, map[string]int{"games": len(owned), "new": res.NewGames}); err != nil {
		return res, err
	}
	if res.FirstImport {
		settings.FirstSyncAt = stamp
	}
	settings.LastSyncAt = stamp
	if err := store.WriteSettings(tx, settings); err != nil {
		return res, err
	}
	return res, tx.Commit()
}

func (s *Service) addGame(tx *sql.Tx, o steam.Owned, now time.Time, stamp string, firstImport bool, classifyDays int) error {
	status, classified := ForNewGame(o.Minutes), true
	if firstImport && o.Minutes > 0 {
		status, classified = ClassifyDefault(o.LastPlayed, now, classifyDays), false
	}
	if _, err := tx.Exec(`INSERT INTO game (appid, name, status, classified, first_seen_at) VALUES (?, ?, ?, ?, ?)`,
		o.AppID, o.Name, status, classified, stamp); err != nil {
		return err
	}
	if !classified {
		return nil // the status is only staged until Save all (L8)
	}
	return changeStatus(tx, now, o.AppID, "", status, "import")
}

func loadExisting(tx *sql.Tx) (map[int]existing, error) {
	rows, err := tx.Query(`SELECT g.appid, g.status, g.classified, COALESCE(s.minutes_total, 0), COALESCE(s.last_played_at, '')
		FROM game g LEFT JOIN latest_snapshot s ON s.appid = g.appid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]existing{}
	for rows.Next() {
		var id int
		var e existing
		if err := rows.Scan(&id, &e.status, &e.classified, &e.minutes, &e.lastPlayed); err != nil {
			return nil, err
		}
		out[id] = e
	}
	return out, rows.Err()
}

func changeStatus(tx events.Execer, now time.Time, appid int, from, to Status, source string) error {
	if from != "" {
		if _, err := tx.Exec(`UPDATE game SET status = ? WHERE appid = ?`, to, appid); err != nil {
			return err
		}
	}
	return events.Log(tx, now, events.StatusChanged, appid, map[string]string{"from": string(from), "to": string(to), "source": source})
}

// State is a game's status fields, returned before a change so the UI can undo it (L3).
type State struct {
	AppID      int     `json:"appid"`
	Status     Status  `json:"status"`
	Completed  bool    `json:"completed"`
	Rating     *string `json:"rating"`
	FinishedAt *string `json:"finishedAt"`
}

// DateFormat is how finished dates are stored: a local calendar date (ADR 0016).
const DateFormat = "2006-01-02"

// SetStatus changes the status of each game and returns their previous states.
// Moving to Finished dates the finish today; leaving Finished clears the rating,
// Completed and the finished date (L4, ADR 0016).
func (s *Service) SetStatus(appids []int, status Status) ([]State, error) {
	if !status.Valid() {
		return nil, errors.New("invalid status")
	}
	now := s.Now()
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var before []State
	for _, id := range appids {
		st, err := readState(tx, id)
		if err != nil {
			return nil, err
		}
		before = append(before, st)
		if st.Status == status {
			continue
		}
		if err := changeStatus(tx, now, id, st.Status, status, "user"); err != nil {
			return nil, err
		}
		q, args := `UPDATE game SET finished_at = ? WHERE appid = ?`, []any{now.Format(DateFormat), id}
		if status != Finished {
			q, args = `UPDATE game SET rating = NULL, completed = 0, finished_at = NULL WHERE appid = ?`, []any{id}
		}
		if _, err := tx.Exec(q, args...); err != nil {
			return nil, err
		}
	}
	return before, tx.Commit()
}

// Restore puts games back to earlier states (the undo toast).
func (s *Service) Restore(states []State) error {
	now := s.Now()
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, want := range states {
		cur, err := readState(tx, want.AppID)
		if err != nil {
			return err
		}
		if cur.Status != want.Status {
			if err := changeStatus(tx, now, want.AppID, cur.Status, want.Status, "undo"); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`UPDATE game SET completed = ?, rating = ?, finished_at = ? WHERE appid = ?`,
			want.Completed, want.Rating, want.FinishedAt, want.AppID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetRating sets or clears (nil) the S-F rating of a Finished game (L4, ADR 0020).
func (s *Service) SetRating(appid int, rating *string) error {
	if rating != nil && !ValidRating(*rating) {
		return errors.New("rating must be S, A, B, C, D or F")
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE game SET rating = ? WHERE appid = ? AND status = ?`, rating, appid, Finished)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("only Finished games take a rating")
	}
	if err := events.Log(tx, s.Now(), events.Rated, appid, map[string]*string{"rating": rating}); err != nil {
		return err
	}
	return tx.Commit()
}

// SetCompleted ticks or clears the Completed box of a Finished game (L1).
func (s *Service) SetCompleted(appid int, completed bool) error {
	res, err := s.DB.Exec(`UPDATE game SET completed = ? WHERE appid = ? AND status = ?`, completed, appid, Finished)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("only Finished games can be marked Completed")
	}
	return nil
}

// SetFinishedAt sets the date a Finished game was finished, for backfilling; "" means
// unknown. Dates in the future are refused (ADR 0016).
func (s *Service) SetFinishedAt(appid int, date string) error {
	var value any
	if date != "" {
		d, err := time.Parse(DateFormat, date)
		if err != nil {
			return errors.New("the finished date must look like 2026-11-06")
		}
		if d.Format(DateFormat) > s.Now().Format(DateFormat) {
			return errors.New("the finished date can't be in the future")
		}
		value = date
	}
	res, err := s.DB.Exec(`UPDATE game SET finished_at = ? WHERE appid = ? AND status = ?`, value, appid, Finished)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("only Finished games have a finished date")
	}
	return nil
}

func readState(tx *sql.Tx, appid int) (State, error) {
	st := State{AppID: appid}
	var rating, finished sql.NullString
	err := tx.QueryRow(`SELECT status, completed, rating, finished_at FROM game WHERE appid = ?`, appid).
		Scan(&st.Status, &st.Completed, &rating, &finished)
	if errors.Is(err, sql.ErrNoRows) {
		return st, errors.New("game not found")
	}
	st.Rating = strPtr(rating)
	if finished.Valid {
		st.FinishedAt = &finished.String
	}
	return st, err
}
