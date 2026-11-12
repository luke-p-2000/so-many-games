package recommend

import (
	"math"
	"strings"
	"testing"
	"time"

	"somanygames/internal/library"
)

func ip(n int) *int { return &n }

func game(id int, st library.Status, minutes int, tags ...string) library.Game {
	w := map[string]int{}
	for i, t := range tags {
		w[t] = 100 - i
	}
	return library.Game{AppID: id, Name: "g", Status: st, Classified: true, Minutes: minutes, Tags: tags, TagWeights: w}
}

func reviewed(g library.Game, pct, count int) library.Game {
	g.ReviewPct, g.ReviewCount, g.HasMeta = ip(pct), ip(count), true
	return g
}

func TestEligiblePool(t *testing.T) {
	limit := Limits{Shelved: 10, Endless: 3}
	cases := []struct {
		name string
		g    library.Game
		want bool
	}{
		{"unplayed", game(1, library.Unplayed, 0), true},
		{"to play with hours", game(2, library.ToPlay, 50*60), true},
		{"shelved under limit", game(3, library.Shelved, 9*60), true},
		{"shelved at limit", game(4, library.Shelved, 10*60), false},
		{"endless under limit", game(5, library.Endless, 60), true},
		{"endless over its own limit", game(6, library.Endless, 5*60), false},
		{"shelved at endless limit", game(13, library.Shelved, 5*60), true},
		{"playing few hours", game(7, library.Playing, 10), false},
		{"finished", game(8, library.Finished, 0), false},
		{"not for me", game(9, library.NotForMe, 0), false},
		{"dead", game(12, library.Dead, 0), false},
	}
	for _, c := range cases {
		if got := Eligible(c.g, limit); got != c.want {
			t.Errorf("%s: Eligible = %v, want %v", c.name, got, c.want)
		}
	}
	hidden := game(10, library.Unplayed, 0)
	hidden.Hidden = true
	unclassified := game(11, library.Shelved, 0)
	unclassified.Classified = false
	if Eligible(hidden, limit) || Eligible(unclassified, limit) {
		t.Error("hidden and unclassified games must never be eligible")
	}
}

func TestWilson(t *testing.T) {
	if Wilson(100, 0) != 0 {
		t.Error("no reviews should score 0")
	}
	// 95% of 10,000 beats 100% of 10: confidence matters.
	if !(Wilson(95, 10000) > Wilson(100, 10)) {
		t.Errorf("wilson(95,10000)=%f wilson(100,10)=%f", Wilson(95, 10000), Wilson(100, 10))
	}
	// Known value: p=0.9, n=100 -> about 0.8256.
	if got := Wilson(90, 100); math.Abs(got-0.8256) > 0.001 {
		t.Errorf("Wilson(90,100) = %f", got)
	}
}

func TestBuildProfileDampensStoplistsAndPins(t *testing.T) {
	games := []library.Game{
		game(1, library.Playing, 1000*60, "Automation", "Singleplayer"), // sqrt(1000) ~ 31.6
		game(2, library.Shelved, 100*60, "Roguelike"),                   // 10 each
		game(3, library.Shelved, 100*60, "Roguelike"),                   // Roguelike total 20
		game(4, library.Shelved, 1*60, "Puzzle", "Cozy", "Racing", "Horror"),
	}
	p := BuildProfile(games, []string{"singleplayer"}, []string{"Metroidvania"}, nil, nil)
	if p.Tags[0].Name != "Automation" || p.Tags[1].Name != "Roguelike" {
		t.Fatalf("order = %+v", p.Tags)
	}
	for _, tg := range p.Tags {
		if tg.Name == "Singleplayer" {
			t.Fatal("stoplisted tag in profile")
		}
	}
	if len(p.Tags) != ProfileSize+1 || !p.Tags[len(p.Tags)-1].Pinned {
		t.Fatalf("expected top 5 plus the pinned tag: %+v", p.Tags)
	}
}

func TestBuildProfileWeighsRatingsAndNotForMe(t *testing.T) {
	rated := func(id int, rating, tag string) library.Game {
		g := game(id, library.Finished, 100*60, tag)
		g.Rating = &rating
		return g
	}
	games := []library.Game{
		rated(1, "S", "Loved"),                         // 10 * 1.75 = 17.5
		game(2, library.Finished, 100*60, "Unrated"),   // 10
		rated(3, "D", "Disliked"),                      // 10 * 0.3 = 3
		game(4, library.NotForMe, 1000*60, "Rejected"), // left out
		game(5, library.Shelved, 25*60, "Played"),      // 5
	}
	p := BuildProfile(games, nil, nil, nil, nil)
	got := []string{}
	for _, tg := range p.Tags {
		got = append(got, tg.Name)
	}
	if strings.Join(got, ",") != "Loved,Unrated,Played,Disliked" {
		t.Fatalf("profile order = %v", got)
	}
}

func TestBuildProfileRecentUsesLastPlayed(t *testing.T) {
	now := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	recent, old := now.AddDate(0, -1, 0), now.AddDate(-2, 0, 0)
	a := game(1, library.Playing, 60*60, "Roguelike")
	a.LastPlayed = &recent
	b := game(2, library.Shelved, 500*60, "Automation")
	b.LastPlayed = &old
	since := RecentSince(now, 6)
	p := BuildProfile([]library.Game{a, b}, nil, nil, nil, &since)
	if len(p.Tags) != 1 || p.Tags[0].Name != "Roguelike" {
		t.Fatalf("recent profile = %+v", p.Tags)
	}
}

func TestTagEngine(t *testing.T) {
	p := Profile{Tags: []WeightedTag{{Name: "Roguelike", Weight: 20}, {Name: "Automation", Weight: 10}}, Avoided: []string{"Horror"}}
	pool := []library.Game{
		reviewed(game(1, library.Unplayed, 0, "Roguelike"), 90, 1000),
		reviewed(game(2, library.Unplayed, 0, "Roguelike"), 98, 60),
		reviewed(game(3, library.Unplayed, 0, "Roguelike"), 99, 49),             // under 50 reviews
		reviewed(game(4, library.Unplayed, 0, "Roguelike", "Horror"), 95, 5000), // avoided
		reviewed(game(5, library.Unplayed, 0, "Puzzle"), 95, 5000),              // no match
		reviewed(game(6, library.Unplayed, 0, "Automation", "Roguelike"), 80, 500),
		game(7, library.Unplayed, 0, "Roguelike"), // no store data
	}
	ranked := TagEngine{MinReviews: 50}.Recommend(p, pool)
	ids := []int{}
	for _, r := range ranked {
		ids = append(ids, r.Game.AppID)
	}
	// 98% of 60 (lower bound ~0.906) outranks 90% of 1,000 (~0.880).
	if len(ids) != 3 || ids[0] != 2 || ids[1] != 1 || ids[2] != 6 {
		t.Fatalf("ranked ids = %v", ids)
	}
	if ranked[2].Row != "Automation" || len(ranked[2].Matched) != 2 {
		t.Fatalf("game 6 should sit in its strongest tag's row: %+v", ranked[2])
	}

	rows := Rows(p, ranked)
	if len(rows) != 2 || rows[0].Tag != "Roguelike" || len(rows[0].Games) != 2 || rows[1].Games[0].Game.AppID != 6 {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestTailoredPrefersMostMatchedTags(t *testing.T) {
	r := func(id int, score float64, matched ...string) Ranked {
		return Ranked{Game: library.Game{AppID: id}, Score: score, Matched: matched}
	}
	// Ranked arrives best score first.
	ranked := []Ranked{
		r(1, 0.95, "Roguelike"),
		r(2, 0.90, "Roguelike", "Deckbuilding"),
		r(3, 0.85, "Roguelike", "Deckbuilding", "Strategy"),
		r(4, 0.80, "Roguelike", "Strategy"),
	}
	ids := []int{}
	for _, x := range Tailored(ranked) {
		ids = append(ids, x.Game.AppID)
	}
	// Three matches first, then the two-match games best score first; one match never qualifies.
	if len(ids) != 3 || ids[0] != 3 || ids[1] != 2 || ids[2] != 4 {
		t.Fatalf("tailored = %v", ids)
	}
}

func TestPickBackUpIgnoresHoursLimit(t *testing.T) {
	now := time.Now()
	older, newer := now.AddDate(-1, 0, 0), now.AddDate(0, -1, 0)
	a := game(1, library.Shelved, 300*60, "Roguelike")
	a.LastPlayed = &newer
	b := game(2, library.Endless, 900*60)
	b.LastPlayed = &older
	c := game(3, library.Playing, 5)
	got := PickBackUp([]library.Game{a, b, c}, Profile{Tags: []WeightedTag{{Name: "Roguelike"}}})
	if len(got) != 2 || got[0].Game.AppID != 2 || got[1].Game.AppID != 1 || len(got[1].Matched) != 1 {
		t.Fatalf("pick back up = %+v", got)
	}
}
