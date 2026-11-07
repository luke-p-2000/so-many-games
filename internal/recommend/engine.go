package recommend

import (
	"sort"
	"strings"

	"somanygames/internal/library"
)

// Ranked is a recommended game with its score and reasons (R6).
type Ranked struct {
	Game    library.Game `json:"game"`
	Score   float64      `json:"score"`
	Matched []string     `json:"matched"` // profile tags the game shares
	Row     string       `json:"row"`     // the profile tag it is grouped under
}

// Engine turns a profile and the eligible pool into ranked games (R6). Future AI or
// local-model engines implement the same interface.
type Engine interface {
	Recommend(p Profile, pool []library.Game) []Ranked
}

// TagEngine is the built-in engine: tag match, minimum reviews, Wilson ranking (R4).
type TagEngine struct {
	MinReviews int
}

func (e TagEngine) Recommend(p Profile, pool []library.Game) []Ranked {
	avoided := lowerSet(p.Avoided)
	profileWeight := map[string]float64{}
	for _, t := range p.Tags {
		profileWeight[strings.ToLower(t.Name)] = t.Weight
	}

	var out []Ranked
	for _, g := range pool {
		if g.ReviewCount == nil || g.ReviewPct == nil || *g.ReviewCount < e.MinReviews {
			continue
		}
		tags := g.TasteTags()
		if hasAny(tags, avoided) {
			continue
		}
		var matched []string
		for _, t := range tags {
			if _, ok := profileWeight[strings.ToLower(t)]; ok {
				matched = append(matched, t)
			}
		}
		if len(matched) == 0 {
			continue
		}
		out = append(out, Ranked{
			Game:    g,
			Score:   Wilson(*g.ReviewPct, *g.ReviewCount),
			Matched: matched,
			Row:     bestRow(g, matched, profileWeight),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// bestRow picks the matched tag the game is most strongly tagged with, breaking ties by
// profile weight, so a game matching several tags appears once in its best-fitting row (R5).
func bestRow(g library.Game, matched []string, profileWeight map[string]float64) string {
	best := matched[0]
	for _, t := range matched[1:] {
		wt, wb := g.TagWeights[t], g.TagWeights[best]
		if wt > wb || (wt == wb && profileWeight[strings.ToLower(t)] > profileWeight[strings.ToLower(best)]) {
			best = t
		}
	}
	return best
}

func hasAny(tags []string, set map[string]bool) bool {
	for _, t := range tags {
		if set[strings.ToLower(t)] {
			return true
		}
	}
	return false
}

// Row is one "Because you play ..." row.
type Row struct {
	Tag   string   `json:"tag"`
	Games []Ranked `json:"games"`
}

// RowDepth is how many ranked games each section holds. A section shows as many as fit on
// one line (at most 6, R5), and its refresh button moves on to the next ones.
const RowDepth = 30

// Rows groups ranked games by their row tag, in profile order, best score first.
func Rows(p Profile, ranked []Ranked) []Row {
	byTag := map[string][]Ranked{}
	for _, r := range ranked {
		key := strings.ToLower(r.Row)
		if len(byTag[key]) < RowDepth {
			byTag[key] = append(byTag[key], r)
		}
	}
	var rows []Row
	for _, t := range p.Tags {
		if games := byTag[strings.ToLower(t.Name)]; len(games) > 0 {
			rows = append(rows, Row{Tag: t.Name, Games: games})
		}
	}
	return rows
}

// MinTailoredMatch is how many profile tags a game must share to be Tailored for you.
const MinTailoredMatch = 2

// Tailored ranks the games that share the most profile tags, best score first among equal
// counts, for the top "Tailored for you" section. These games also stay in their tag rows;
// the frontend leaves the ones Tailored for you is showing out of those rows.
func Tailored(ranked []Ranked) []Ranked {
	var out []Ranked
	for _, r := range ranked {
		if len(r.Matched) >= MinTailoredMatch {
			out = append(out, r)
		}
	}
	// Stable over the score order, so equal counts stay best score first.
	sort.SliceStable(out, func(i, j int) bool { return len(out[i].Matched) > len(out[j].Matched) })
	if len(out) > RowDepth {
		out = out[:RowDepth]
	}
	return out
}
