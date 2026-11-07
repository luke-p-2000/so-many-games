// Package recommend builds the taste profile (R1, R2), the eligible pool shared with the
// wheel (R3) and ranks candidates through a swappable Engine (R4–R6).
package recommend

import (
	"math"
	"sort"
	"strings"
	"time"

	"somanygames/internal/library"
)

// Eligible is the single pool rule used by recommendations, the wheel and the backlog
// (R3, ADR 0008): classified, not hidden, Unplayed or To play at any hours, or Shelved or
// Endless under the hours limit.
func Eligible(g library.Game, hoursLimit int) bool {
	if g.Hidden || !g.Classified {
		return false
	}
	switch g.Status {
	case library.Unplayed, library.ToPlay:
		return true
	case library.Shelved, library.Endless:
		return g.Hours() < float64(hoursLimit)
	}
	return false
}

// Pool returns the eligible games.
func Pool(games []library.Game, hoursLimit int) []library.Game {
	var out []library.Game
	for _, g := range games {
		if Eligible(g, hoursLimit) {
			out = append(out, g)
		}
	}
	return out
}

// WeightedTag is one profile tag and its dampened hours.
type WeightedTag struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
	Pinned bool    `json:"pinned"`
}

// Profile is the engine's view of the owner's taste.
type Profile struct {
	Tags    []WeightedTag `json:"tags"`
	Avoided []string      `json:"avoided"`
}

// ProfileSize is the number of tags taken from play history (R1).
const ProfileSize = 5

// BuildProfile ranks tags by square-root-dampened hours so one very long game cannot
// dominate, scaled by how much the owner liked the game (ADR 0015). It drops stoplisted
// tags, keeps the top five and adds pinned tags (R1, R2).
// A non-nil since limits it to games last played after that time (the Recent view, ADR 0009).
func BuildProfile(games []library.Game, stoplist, pinned, avoided []string, since *time.Time) Profile {
	stop := lowerSet(stoplist)
	sum := map[string]float64{}
	for _, g := range games {
		if g.Hidden || g.Minutes == 0 {
			continue
		}
		if since != nil && (g.LastPlayed == nil || g.LastPlayed.Before(*since)) {
			continue
		}
		w := math.Sqrt(g.Hours()) * liking(g)
		if w == 0 {
			continue
		}
		for _, t := range g.TasteTags() {
			if !stop[strings.ToLower(t)] {
				sum[t] += w
			}
		}
	}
	tags := make([]WeightedTag, 0, len(sum))
	for name, w := range sum {
		tags = append(tags, WeightedTag{Name: name, Weight: w})
	}
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Weight != tags[j].Weight {
			return tags[i].Weight > tags[j].Weight
		}
		return tags[i].Name < tags[j].Name
	})
	if len(tags) > ProfileSize {
		tags = tags[:ProfileSize]
	}

	have := map[string]bool{}
	top := 1.0
	for _, t := range tags {
		have[strings.ToLower(t.Name)] = true
		top = math.Max(top, t.Weight)
	}
	for _, p := range pinned {
		if !have[strings.ToLower(p)] {
			tags = append(tags, WeightedTag{Name: p, Weight: top, Pinned: true})
		}
	}
	return Profile{Tags: tags, Avoided: avoided}
}

// NeutralRating is the rating that leaves a game's weight unchanged.
const NeutralRating = 5.5

// liking scales a game's taste weight (ADR 0015): Not for me counts for nothing, a rating
// scales around NeutralRating (10 counts about double, 1 under a fifth), and anything
// else counts as played.
func liking(g library.Game) float64 {
	if g.Status == library.NotForMe {
		return 0
	}
	if g.Rating != nil {
		return float64(*g.Rating) / NeutralRating
	}
	return 1
}

// RecentSince is the start of the Recent window.
func RecentSince(now time.Time, months int) time.Time {
	return now.AddDate(0, -months, 0)
}

// Wilson is the lower bound of the 95% Wilson score interval for a positive-review share.
func Wilson(pct, count int) float64 {
	if count <= 0 {
		return 0
	}
	const z = 1.96
	n := float64(count)
	p := float64(pct) / 100
	return (p + z*z/(2*n) - z*math.Sqrt(p*(1-p)/n+z*z/(4*n*n))) / (1 + z*z/n)
}

func lowerSet(list []string) map[string]bool {
	m := map[string]bool{}
	for _, s := range list {
		m[strings.ToLower(strings.TrimSpace(s))] = true
	}
	return m
}
