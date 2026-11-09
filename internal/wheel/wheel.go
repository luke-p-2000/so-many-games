// Package wheel narrows the eligible pool with filters, draws the wheel and picks a winner (W1, W2).
package wheel

import (
	"math/rand/v2"
	"slices"
	"strings"

	"somanygames/internal/library"
)

// Player types for the Type filter; "" means All.
var Types = []string{"single-player", "co-op", "pvp"}

// Filters narrow the pool by AND. Every zero value means All (W1).
type Filters struct {
	Tags         []string `json:"tags"` // a game must have every one
	Genre        string   `json:"genre"`
	MinReviewPct int      `json:"minReviewPct"`
	Type         string   `json:"type"` // one of Types: games with that Steam player category
	Status       []string `json:"status"` // games with any of these statuses; none means all but Endless
}

// Filter returns the pool games that pass every filter. Endless games only join the wheel
// when the Endless status is ticked, since they rarely end.
func Filter(pool []library.Game, f Filters) []library.Game {
	var out []library.Game
	for _, g := range pool {
		if !hasAll(g.TasteTags(), f.Tags) {
			continue
		}
		if f.Genre != "" && !containsFold(g.Genres, f.Genre) {
			continue
		}
		if f.MinReviewPct > 0 && (g.ReviewPct == nil || *g.ReviewPct < f.MinReviewPct) {
			continue
		}
		if f.Type != "" && !slices.Contains(g.Categories, f.Type) {
			continue
		}
		if len(f.Status) > 0 && !slices.Contains(f.Status, string(g.Status)) {
			continue
		}
		if len(f.Status) == 0 && g.Status == library.Endless {
			continue
		}
		out = append(out, g)
	}
	return out
}

func hasAll(list, want []string) bool {
	for _, w := range want {
		if !containsFold(list, w) {
			return false
		}
	}
	return true
}

func containsFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

// Weights are the taste-bias weights by appid (W2); nil means uniform.
type Weights map[int]float64

func (w Weights) of(g library.Game) float64 {
	if w == nil {
		return 1
	}
	// A small floor keeps games with no taste match on the wheel.
	return 0.05 + w[g.AppID]
}

// Draw fills the wheel with up to size games (0 means All), favouring higher weights.
func Draw(games []library.Game, size int, w Weights, rng *rand.Rand) []library.Game {
	n := len(games)
	if size > 0 && size < n {
		n = size
	}
	remaining := slices.Clone(games)
	var entries []library.Game
	for len(entries) < n {
		i := Pick(remaining, w, rng)
		entries = append(entries, remaining[i])
		remaining = slices.Delete(remaining, i, i+1)
	}
	return entries
}

// Pick returns the winning index among entries, or -1 when there are none.
func Pick(entries []library.Game, w Weights, rng *rand.Rand) int {
	if len(entries) == 0 {
		return -1
	}
	total := 0.0
	for _, g := range entries {
		total += w.of(g)
	}
	r := rng.Float64() * total
	for i, g := range entries {
		r -= w.of(g)
		if r < 0 {
			return i
		}
	}
	return len(entries) - 1
}
