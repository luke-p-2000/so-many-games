// Package stats computes the Stats screen (T1). Hidden and unclassified games never count.
package stats

import (
	"database/sql"
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"somanygames/internal/events"
	"somanygames/internal/library"
	"somanygames/internal/recommend"
	"somanygames/internal/store"
)

// Bar is one labelled value.
type Bar struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// Stats is everything the Stats screen shows.
type Stats struct {
	StatusCounts     []Bar   `json:"statusCounts"`
	BacklogSize      int     `json:"backlogSize"`
	BacklogHours     float64 `json:"backlogHours"`
	TagHoursAll      []Bar   `json:"tagHoursAll"`
	TagHoursRecent   []Bar   `json:"tagHoursRecent"`
	GenreHoursAll    []Bar   `json:"genreHoursAll"`
	GenreHoursRecent []Bar   `json:"genreHoursRecent"`
	FinishedPerYear  []Bar   `json:"finishedPerYear"`
	Ratings          []Bar   `json:"ratings"`
	MostPlayed       []Bar   `json:"mostPlayed"`
	BurnDown         []Bar   `json:"burnDown"`
	FirstSyncAt      string  `json:"firstSyncAt"`
}

// BeforeFirstSync labels Finished games classified on import (ADR 0007).
const BeforeFirstSync = "Before first sync"

const topN = 8

// Compute builds the stats from the games and the event log.
func Compute(db *sql.DB, games []library.Game, s store.Settings, now time.Time) (Stats, error) {
	var counted []library.Game
	for _, g := range games {
		if !g.Hidden && g.Classified {
			counted = append(counted, g)
		}
	}
	out := Stats{FirstSyncAt: s.FirstSyncAt}

	byStatus := map[library.Status]int{}
	for _, g := range counted {
		byStatus[g.Status]++
	}
	for _, st := range library.Statuses {
		out.StatusCounts = append(out.StatusCounts, Bar{string(st), float64(byStatus[st])})
	}

	out.BacklogSize, out.BacklogHours = backlog(counted, limits(s))

	since := recommend.RecentSince(now, s.RecentMonths)
	out.TagHoursAll = hoursBy(counted, nil, func(g library.Game) []string { return g.Tags })
	out.TagHoursRecent = hoursBy(counted, &since, func(g library.Game) []string { return g.Tags })
	out.GenreHoursAll = hoursBy(counted, nil, func(g library.Game) []string { return g.Genres })
	out.GenreHoursRecent = hoursBy(counted, &since, func(g library.Game) []string { return g.Genres })

	ratings := make([]int, 11)
	for _, g := range counted {
		if g.Status == library.Finished && g.Rating != nil {
			ratings[*g.Rating]++
		}
	}
	for r := 1; r <= 10; r++ {
		if ratings[r] > 0 {
			out.Ratings = append(out.Ratings, Bar{strconv.Itoa(r), float64(ratings[r])})
		}
	}

	played := append([]library.Game(nil), counted...)
	sort.Slice(played, func(i, j int) bool { return played[i].Minutes > played[j].Minutes })
	for _, g := range played[:min(5, len(played))] {
		if g.Minutes > 0 {
			out.MostPlayed = append(out.MostPlayed, Bar{g.Name, g.Hours()})
		}
	}

	var err error
	if out.FinishedPerYear, err = finishedPerYear(db, counted); err != nil {
		return out, err
	}
	out.BurnDown, err = burnDown(db)
	return out, err
}

func hoursBy(games []library.Game, since *time.Time, keys func(library.Game) []string) []Bar {
	sum := map[string]float64{}
	for _, g := range games {
		if since != nil && (g.LastPlayed == nil || g.LastPlayed.Before(*since)) {
			continue
		}
		for _, k := range keys(g) {
			sum[k] += g.Hours()
		}
	}
	bars := make([]Bar, 0, len(sum))
	for k, v := range sum {
		if v > 0 {
			bars = append(bars, Bar{k, v})
		}
	}
	sort.Slice(bars, func(i, j int) bool { return bars[i].Value > bars[j].Value })
	return bars[:min(topN, len(bars))]
}

// finishedPerYear counts currently Finished games by the year of their latest move to
// Finished; moves made on the Classify screen or at import count as before first sync.
func finishedPerYear(db *sql.DB, games []library.Game) ([]Bar, error) {
	finished := map[int]bool{}
	for _, g := range games {
		if g.Status == library.Finished {
			finished[g.AppID] = true
		}
	}
	rows, err := db.Query(`SELECT appid, at, payload FROM event WHERE kind = ? ORDER BY at`, events.StatusChanged)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	latest := map[int]string{}
	for rows.Next() {
		var id int
		var at, payload string
		if err := rows.Scan(&id, &at, &payload); err != nil {
			return nil, err
		}
		var p struct{ To, Source string }
		if json.Unmarshal([]byte(payload), &p) != nil || p.To != string(library.Finished) || !finished[id] {
			continue
		}
		if p.Source == "classify" || p.Source == "import" {
			latest[id] = BeforeFirstSync
		} else if len(at) >= 4 {
			latest[id] = at[:4]
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	count := map[string]float64{}
	for _, label := range latest {
		count[label]++
	}
	var bars []Bar
	if n := count[BeforeFirstSync]; n > 0 {
		bars = append(bars, Bar{BeforeFirstSync, n})
		delete(count, BeforeFirstSync)
	}
	years := make([]string, 0, len(count))
	for y := range count {
		years = append(years, y)
	}
	sort.Strings(years)
	for _, y := range years {
		bars = append(bars, Bar{y, count[y]})
	}
	return bars, nil
}

// burnDown is the backlog size at each sync day, from backlog_snapshot events (ADR 0008).
func burnDown(db *sql.DB) ([]Bar, error) {
	rows, err := db.Query(`SELECT substr(at, 1, 10), payload FROM event WHERE kind = ? ORDER BY at`, events.BacklogSnapshot)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bars []Bar
	for rows.Next() {
		var day, payload string
		if err := rows.Scan(&day, &payload); err != nil {
			return nil, err
		}
		var p struct{ Size int }
		if json.Unmarshal([]byte(payload), &p) != nil {
			continue
		}
		if n := len(bars); n > 0 && bars[n-1].Label == day {
			bars[n-1].Value = float64(p.Size) // keep the last snapshot of each day
		} else {
			bars = append(bars, Bar{day, float64(p.Size)})
		}
	}
	return bars, rows.Err()
}

// LogBacklog records today's backlog for the burn-down; called after each sync.
func LogBacklog(db *sql.DB, games []library.Game, s store.Settings, now time.Time) error {
	size, hours := backlog(games, limits(s))
	return events.Log(db, now, events.BacklogSnapshot, 0, map[string]any{"size": size, "hours": hours})
}

// backlog is the eligible pool's size and played hours (ADR 0008).
func backlog(games []library.Game, l recommend.Limits) (size int, hours float64) {
	pool := recommend.Pool(games, l)
	for _, g := range pool {
		hours += g.Hours()
	}
	return len(pool), hours
}

func limits(s store.Settings) recommend.Limits {
	return recommend.Limits{Shelved: s.ShelvedHours, Endless: s.EndlessHours}
}
