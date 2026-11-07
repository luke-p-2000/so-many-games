package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
)

// Settings holds every user preference (G1). Each field is one row in the setting table,
// keyed by its JSON name. The API key is deliberately absent (S2).
type Settings struct {
	SteamID        string   `json:"steam_id"`
	ProfileURL     string   `json:"profile_url"`
	HoursLimit     int      `json:"hours_limit"`     // Shelved/Endless eligible under this many hours (R3)
	MinReviews     int      `json:"min_reviews"`     // R4, ADR 0010
	RecentMonths   int      `json:"recent_months"`   // R1 Recent view
	AbandonedDays  int      `json:"abandoned_days"`  // N1
	ClassifyDays   int      `json:"classify_days"`   // L7 split
	WheelSize      int      `json:"wheel_size"`      // 12, 15, 18, 24, or 0 for All (W2)
	Stoplist       []string `json:"stoplist"`        // R1
	Pinned         []string `json:"pinned"`          // R2
	Avoided        []string `json:"avoided"`         // R2
	HiddenPatterns []string `json:"hidden_patterns"` // L6
	FirstSyncAt    string   `json:"first_sync_at"`
	LastSyncAt     string   `json:"last_sync_at"`
}

// WheelSizes are the allowed wheel sizes; 0 means All.
var WheelSizes = []int{12, 15, 18, 24, 0}

// DefaultSettings are the values from the requirements.
func DefaultSettings() Settings {
	return Settings{
		HoursLimit:    10,
		MinReviews:    50,
		RecentMonths:  6,
		AbandonedDays: 60,
		ClassifyDays:  30,
		WheelSize:     12,
		Stoplist:      []string{"Singleplayer", "Multiplayer", "Indie", "Action", "Adventure", "Casual", "2D"},
		Pinned:        []string{},
		Avoided:       []string{},
		HiddenPatterns: []string{
			"Test Server", "Public Test", "Playtest", "Unstable", "Beta",
			"Dedicated Server", "SDK", "Soundtrack", "Demo",
		},
	}
}

// LoadSettings returns stored settings over the defaults.
func LoadSettings(db *sql.DB) (Settings, error) {
	s := DefaultSettings()
	rows, err := db.Query(`SELECT key, value FROM setting`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	stored := map[string]json.RawMessage{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return s, err
		}
		stored[k] = json.RawMessage(v)
	}
	if err := rows.Err(); err != nil {
		return s, err
	}
	// Round-trip through a map so absent keys keep their defaults.
	merged, err := toMap(s)
	if err != nil {
		return s, err
	}
	for k, v := range stored {
		merged[k] = v
	}
	b, err := json.Marshal(merged)
	if err != nil {
		return s, err
	}
	return s, json.Unmarshal(b, &s)
}

// SaveSettings writes every field in its own transaction.
func SaveSettings(db *sql.DB, s Settings) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := WriteSettings(tx, s); err != nil {
		return err
	}
	return tx.Commit()
}

// WriteSettings writes every field inside the caller's transaction, so settings change
// together with the data they belong to.
func WriteSettings(tx *sql.Tx, s Settings) error {
	if !slices.Contains(WheelSizes, s.WheelSize) {
		return errors.New("wheel size must be 12, 15, 18, 24 or All")
	}
	if s.HoursLimit < 0 || s.MinReviews < 0 || s.RecentMonths < 1 || s.AbandonedDays < 1 || s.ClassifyDays < 1 {
		return errors.New("settings values must be positive")
	}
	m, err := toMap(s)
	if err != nil {
		return err
	}
	for k, v := range m {
		if _, err := tx.Exec(`INSERT INTO setting (key, value) VALUES (?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value`, k, string(v)); err != nil {
			return err
		}
	}
	return nil
}

func toMap(s Settings) (map[string]json.RawMessage, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	m := map[string]json.RawMessage{}
	return m, json.Unmarshal(b, &m)
}
