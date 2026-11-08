package library

import (
	"database/sql"
	"fmt"
	"slices"
)

// Tiers are the tier list rows, best first (T2).
var Tiers = []string{"S", "A", "B", "C", "D", "F"}

// TierEntry places one game in the tier list.
type TierEntry struct {
	AppID    int    `json:"appid"`
	Tier     string `json:"tier"`
	Position int    `json:"position"`
}

// LoadTiers returns every placement in tier order, then position.
func LoadTiers(db *sql.DB) ([]TierEntry, error) {
	rows, err := db.Query(`SELECT appid, tier, position FROM tier_entry ORDER BY tier, position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TierEntry{}
	for rows.Next() {
		var e TierEntry
		if err := rows.Scan(&e.AppID, &e.Tier, &e.Position); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// SaveTiers replaces the whole tier list. The screen sends every placement after each drop,
// including those of games no longer shown, so they keep their spot.
func SaveTiers(db *sql.DB, entries []TierEntry) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := WriteTiers(tx, entries); err != nil {
		return err
	}
	return tx.Commit()
}

// WriteTiers replaces the tier list inside the caller's transaction (also used by import).
func WriteTiers(tx *sql.Tx, entries []TierEntry) error {
	if _, err := tx.Exec(`DELETE FROM tier_entry`); err != nil {
		return err
	}
	for _, e := range entries {
		if !slices.Contains(Tiers, e.Tier) {
			return fmt.Errorf("unknown tier %q", e.Tier)
		}
		// Placements for games no longer in the library are dropped rather than failing.
		if _, err := tx.Exec(`INSERT OR REPLACE INTO tier_entry (appid, tier, position)
			SELECT appid, ?, ? FROM game WHERE appid = ?`, e.Tier, e.Position, e.AppID); err != nil {
			return err
		}
	}
	return nil
}
