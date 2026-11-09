package wheel

import (
	"math/rand/v2"
	"testing"

	"somanygames/internal/library"
)

func g(id int, pct int, cats []string, tags ...string) library.Game {
	p := pct
	return library.Game{AppID: id, HasMeta: true, ReviewPct: &p, Categories: cats, Tags: tags, Genres: []string{"Action"}}
}

func TestFilterDefaultsToAll(t *testing.T) {
	pool := []library.Game{
		g(1, 95, []string{"single-player"}, "Roguelike", "Deckbuilding"),
		g(2, 70, []string{"co-op", "pvp"}, "Shooter", "Roguelike"),
		{AppID: 3}, // no store data
	}
	if got := Filter(pool, Filters{}); len(got) != 3 {
		t.Fatalf("unfiltered wheel must draw from the whole pool, got %d", len(got))
	}
	if got := Filter(pool, Filters{Tags: []string{"roguelike"}}); len(got) != 2 {
		t.Fatalf("tag filter = %v", got)
	}
	if got := Filter(pool, Filters{Tags: []string{"Roguelike", "deckbuilding"}}); len(got) != 1 || got[0].AppID != 1 {
		t.Fatalf("several tags must all match: %v", got)
	}
	if got := Filter(pool, Filters{MinReviewPct: 80}); len(got) != 1 {
		t.Fatalf("review filter = %v", got)
	}
	if got := Filter(pool, Filters{Type: "co-op"}); len(got) != 1 || got[0].AppID != 2 {
		t.Fatalf("type filter = %v", got)
	}
	pool[0].Status, pool[1].Status, pool[2].Status = library.Shelved, library.Unplayed, library.Endless
	if got := Filter(pool, Filters{Status: []string{"shelved", "endless"}}); len(got) != 2 || got[0].AppID != 1 || got[1].AppID != 3 {
		t.Fatalf("status filter = %v", got)
	}
	if got := Filter(pool, Filters{}); len(got) != 2 || got[0].AppID != 1 || got[1].AppID != 2 {
		t.Fatalf("Endless must stay off the wheel unless ticked: %v", got)
	}
	if got := Filter(pool, Filters{Type: "single-player", Tags: []string{"Shooter"}}); len(got) != 0 {
		t.Fatalf("filters must AND: %v", got)
	}
}

func TestDrawSize(t *testing.T) {
	var pool []library.Game
	for i := 1; i <= 30; i++ {
		pool = append(pool, library.Game{AppID: i})
	}
	rng := rand.New(rand.NewPCG(1, 2))
	entries := Draw(pool, 12, nil, rng)
	if len(entries) != 12 {
		t.Fatalf("entries=%d", len(entries))
	}
	seen := map[int]bool{}
	for _, e := range entries {
		if seen[e.AppID] {
			t.Fatal("duplicate entry")
		}
		seen[e.AppID] = true
	}
	if all := Draw(pool, 0, nil, rng); len(all) != 30 {
		t.Fatalf("All should use the whole pool, got %d", len(all))
	}
	if w := Pick(nil, nil, rng); w != -1 {
		t.Fatal("an empty wheel has no winner")
	}
	if w := Pick(entries, nil, rng); w < 0 || w >= 12 {
		t.Fatalf("winner=%d", w)
	}
}

func TestBiasFavoursTaste(t *testing.T) {
	pool := []library.Game{{AppID: 1}, {AppID: 2}}
	weights := Weights{1: 0.95}
	rng := rand.New(rand.NewPCG(3, 4))
	wins := 0
	for i := 0; i < 1000; i++ {
		if pool[Pick(pool, weights, rng)].AppID == 1 {
			wins++
		}
	}
	if wins < 850 {
		t.Fatalf("biased game won %d of 1000", wins)
	}
}
