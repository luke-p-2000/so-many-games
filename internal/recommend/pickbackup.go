package recommend

import (
	"sort"
	"strings"

	"somanygames/internal/library"
)

// BackUp is one Pick back up entry with the profile tags it shares (R7).
type BackUp struct {
	Game    library.Game `json:"game"`
	Matched []string     `json:"matched"`
}

// PickBackUp lists every visible, classified Shelved and Endless game at any hours (R7:
// not limited by the pool's hours limit), longest since last played first.
func PickBackUp(games []library.Game, p Profile) []BackUp {
	inProfile := map[string]bool{}
	for _, t := range p.Tags {
		inProfile[strings.ToLower(t.Name)] = true
	}
	var out []BackUp
	for _, g := range games {
		if g.Hidden || !g.Classified || (g.Status != library.Shelved && g.Status != library.Endless) {
			continue
		}
		var matched []string
		for _, t := range g.TasteTags() {
			if inProfile[strings.ToLower(t)] {
				matched = append(matched, t)
			}
		}
		out = append(out, BackUp{Game: g, Matched: matched})
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Game.LastPlayed, out[j].Game.LastPlayed
		switch {
		case a == nil:
			return b != nil
		case b == nil:
			return false
		}
		return a.Before(*b)
	})
	return out
}
