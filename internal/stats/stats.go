// Package stats computes the Stats screen (T1). Hidden and unclassified games never count.
package stats

import (
	"sort"
	"time"

	"somanygames/internal/library"
	"somanygames/internal/recommend"
	"somanygames/internal/store"
)

// Bar is one labelled value.
type Bar struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// Stats is everything the Stats screen shows (T1). The finished counter reads the
// Finished entry of StatusCounts against their total.
type Stats struct {
	StatusCounts     []Bar          `json:"statusCounts"`
	BacklogSize      int            `json:"backlogSize"`
	BacklogHours     float64        `json:"backlogHours"`
	TagHoursAll      []Bar          `json:"tagHoursAll"`
	TagHoursRecent   []Bar          `json:"tagHoursRecent"`
	GenreHoursAll    []Bar          `json:"genreHoursAll"`
	GenreHoursRecent []Bar          `json:"genreHoursRecent"`
	MostPlayed       []library.Game `json:"mostPlayed"`
	LatestFinished   *library.Game  `json:"latestFinished"` // the Finished game with the latest date, if any has one
}

const topN = 8

// Compute builds the stats from the games.
func Compute(games []library.Game, s store.Settings, now time.Time) Stats {
	var counted []library.Game
	for _, g := range games {
		if !g.Hidden && g.Classified {
			counted = append(counted, g)
		}
	}
	var out Stats

	byStatus := map[library.Status]int{}
	for _, g := range counted {
		byStatus[g.Status]++
		if g.Status == library.Finished && g.FinishedAt != nil &&
			(out.LatestFinished == nil || *g.FinishedAt > *out.LatestFinished.FinishedAt) {
			out.LatestFinished = &g
		}
	}
	for _, st := range library.Statuses {
		out.StatusCounts = append(out.StatusCounts, Bar{string(st), float64(byStatus[st])})
	}

	pool := recommend.Pool(counted, recommend.Limits{Shelved: s.ShelvedHours, Endless: s.EndlessHours})
	out.BacklogSize = len(pool)
	for _, g := range pool {
		out.BacklogHours += g.Hours()
	}

	since := recommend.RecentSince(now, s.RecentMonths)
	out.TagHoursAll = hoursBy(counted, nil, func(g library.Game) []string { return g.Tags })
	out.TagHoursRecent = hoursBy(counted, &since, func(g library.Game) []string { return g.Tags })
	out.GenreHoursAll = hoursBy(counted, nil, func(g library.Game) []string { return g.Genres })
	out.GenreHoursRecent = hoursBy(counted, &since, func(g library.Game) []string { return g.Genres })

	played := append([]library.Game(nil), counted...)
	sort.SliceStable(played, func(i, j int) bool { return played[i].Minutes > played[j].Minutes })
	for _, g := range played[:min(5, len(played))] {
		if g.Minutes > 0 {
			out.MostPlayed = append(out.MostPlayed, g)
		}
	}
	return out
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
