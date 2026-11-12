package library

import (
	"database/sql"
	"time"
)

// Game is the read model every screen and the recommender works from.
type Game struct {
	AppID           int            `json:"appid"`
	Name            string         `json:"name"`
	Status          Status         `json:"status"`
	Completed       bool           `json:"completed"`
	Rating          *string        `json:"rating"`
	FinishedAt      *string        `json:"finishedAt"` // YYYY-MM-DD while Finished; nil when unknown (ADR 0016)
	Hidden          bool           `json:"hidden"`
	HiddenReason    string         `json:"hiddenReason"`
	Classified      bool           `json:"classified"`
	MetaUnavailable bool           `json:"metaUnavailable"`
	Minutes         int            `json:"minutes"`
	LastPlayed      *time.Time     `json:"lastPlayed"`
	HasMeta         bool           `json:"hasMeta"`
	ReviewPct       *int           `json:"reviewPct"`
	ReviewCount     *int           `json:"reviewCount"`
	ReleaseDate     string         `json:"releaseDate"`
	Cover           string         `json:"cover"` // file name under the covers folder, or ""
	Tags            []string       `json:"tags"`  // store tags, strongest first
	Genres          []string       `json:"genres"`
	Categories      []string       `json:"categories"` // "single-player", "co-op", "pvp"
	TagWeights      map[string]int `json:"-"`
}

// Hours is lifetime playtime in hours.
func (g Game) Hours() float64 { return float64(g.Minutes) / 60 }

// TasteTags are the tags used for taste and matching: store tags, or genres and
// categories when a game has none (S9).
func (g Game) TasteTags() []string {
	if len(g.Tags) > 0 {
		return g.Tags
	}
	return append(append([]string{}, g.Genres...), g.Categories...)
}

// Load reads every game with its latest playtime and store data.
func Load(db *sql.DB) ([]Game, error) {
	rows, err := db.Query(`
		SELECT g.appid, g.name, g.status, g.completed, g.rating, g.finished_at, g.hidden, g.hidden_reason,
		       g.classified, g.meta_unavailable,
		       COALESCE(s.minutes_total, 0), s.last_played_at,
		       m.appid IS NOT NULL, m.review_pct, m.review_count, COALESCE(m.release_date, ''),
		       COALESCE(m.cover_path, '')
		FROM game g
		LEFT JOIN latest_snapshot s ON s.appid = g.appid
		LEFT JOIN game_meta m ON m.appid = g.appid
		ORDER BY g.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []Game
	index := map[int]int{}
	for rows.Next() {
		var g Game
		var pct, count sql.NullInt64
		var rating sql.NullString
		var last, finished sql.NullString
		if err := rows.Scan(&g.AppID, &g.Name, &g.Status, &g.Completed, &rating, &finished, &g.Hidden, &g.HiddenReason,
			&g.Classified, &g.MetaUnavailable, &g.Minutes, &last,
			&g.HasMeta, &pct, &count, &g.ReleaseDate, &g.Cover); err != nil {
			return nil, err
		}
		g.Rating = strPtr(rating)
		if finished.Valid {
			g.FinishedAt = &finished.String
		}
		g.ReviewPct = intPtr(pct)
		g.ReviewCount = intPtr(count)
		if last.Valid {
			if t, err := time.Parse(time.RFC3339, last.String); err == nil {
				g.LastPlayed = &t
			}
		}
		g.TagWeights = map[string]int{}
		index[g.AppID] = len(games)
		games = append(games, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tags, err := db.Query(`SELECT appid, kind, name, COALESCE(weight, 0) FROM game_tag
		ORDER BY appid, kind, weight DESC, name`)
	if err != nil {
		return nil, err
	}
	defer tags.Close()
	for tags.Next() {
		var appid, weight int
		var kind, name string
		if err := tags.Scan(&appid, &kind, &name, &weight); err != nil {
			return nil, err
		}
		i, ok := index[appid]
		if !ok {
			continue
		}
		g := &games[i]
		switch kind {
		case "tag":
			g.Tags = append(g.Tags, name)
			g.TagWeights[name] = weight
		case "genre":
			g.Genres = append(g.Genres, name)
		case "category":
			g.Categories = append(g.Categories, name)
		}
	}
	return games, tags.Err()
}

func intPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

func strPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}
