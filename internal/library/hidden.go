package library

import (
	"database/sql"
	"errors"
)

// Hide overrides for a single game (L6 Hidden view actions).
const (
	OverrideNone = ""
	OverrideShow = "show" // "Not clutter": never hidden automatically
	OverrideHide = "hide" // hidden by hand
)

// SetHideOverride records a Not clutter / Hide choice and recomputes the hidden flag.
func (s *Service) SetHideOverride(appid int, override string, patterns []string) error {
	if override != OverrideNone && override != OverrideShow && override != OverrideHide {
		return errors.New("invalid hide override")
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE game SET hide_override = ? WHERE appid = ?`, override, appid); err != nil {
		return err
	}
	if err := RecomputeHidden(tx, patterns); err != nil {
		return err
	}
	return tx.Commit()
}

// RecomputeHidden refreshes every game's hidden flag from the name patterns, the store
// type and the per-game override. It runs after syncs, fetches and pattern edits.
func RecomputeHidden(tx *sql.Tx, patterns []string) error {
	rows, err := tx.Query(`SELECT g.appid, g.name, g.hide_override, m.is_game
		FROM game g LEFT JOIN game_meta m ON m.appid = g.appid`)
	if err != nil {
		return err
	}
	reasonFor := hiddenMatcher(patterns)
	type row struct {
		id     int
		hidden bool
		reason string
	}
	var updates []row
	for rows.Next() {
		var id int
		var name, override string
		var isGame sql.NullBool
		if err := rows.Scan(&id, &name, &override, &isGame); err != nil {
			rows.Close()
			return err
		}
		var ig *bool
		if isGame.Valid {
			ig = &isGame.Bool
		}
		r := row{id: id}
		switch override {
		case OverrideHide:
			r.hidden, r.reason = true, "manual"
		case OverrideShow:
		default:
			r.reason = reasonFor(name, ig)
			r.hidden = r.reason != ""
		}
		updates = append(updates, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, u := range updates {
		if _, err := tx.Exec(`UPDATE game SET hidden = ?, hidden_reason = ? WHERE appid = ?`, u.hidden, u.reason, u.id); err != nil {
			return err
		}
	}
	return nil
}
