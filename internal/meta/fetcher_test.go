package meta

import (
	"context"
	"errors"
	"testing"
	"time"

	"somanygames/internal/library"
	"somanygames/internal/steam"
	"somanygames/internal/store"
)

type fakeSource struct {
	items  map[int]Item
	genres map[int][]string
	calls  int
}

func (f *fakeSource) Items(_ context.Context, ids []int) (map[int]Item, error) {
	f.calls++
	out := map[int]Item{}
	for _, id := range ids {
		if it, ok := f.items[id]; ok {
			out[id] = it
		}
	}
	return out, nil
}

func (f *fakeSource) Genres(_ context.Context, id int) ([]string, error) {
	if g, ok := f.genres[id]; ok {
		return g, nil
	}
	return nil, ErrNotFound
}

func (f *fakeSource) Cover(context.Context, string) ([]byte, error) {
	return nil, errors.New("offline")
}

func setup(t *testing.T, src Source) (*Fetcher, *library.Service) {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	now := func() time.Time { return time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC) }
	lib := &library.Service{DB: db, Now: now}
	settings := store.DefaultSettings()
	if _, err := lib.Import([]steam.Owned{
		{AppID: 10, Name: "Good Game"},
		{AppID: 20, Name: "Delisted Game"},
		{AppID: 30, Name: "A Tool"},
	}, settings); err != nil {
		t.Fatal(err)
	}
	return &Fetcher{DB: db, Source: src, CoversDir: t.TempDir(), Now: now}, lib
}

func load(t *testing.T, f *Fetcher, id int) library.Game {
	t.Helper()
	games, err := library.Load(f.DB)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range games {
		if g.AppID == id {
			return g
		}
	}
	t.Fatalf("game %d missing", id)
	return library.Game{}
}

func TestFetchStoresDataRecordsFailuresAndHidesNonGames(t *testing.T) {
	pct, count := 94, 12430
	src := &fakeSource{
		items: map[int]Item{
			10: {Tags: []Tag{{"Roguelike", 900}, {"Action", 500}}, ReviewPct: &pct, ReviewCount: &count,
				ReleaseDate: "2024-05-06", IsGame: true, Categories: []string{"single-player", "co-op"}},
			30: {IsGame: false},
		},
		genres: map[int][]string{10: {"Action", "Indie"}},
	}
	f, _ := setup(t, src)

	if n, _ := f.MissingCount(); n != 3 {
		t.Fatalf("missing before = %d", n)
	}
	ids, _ := f.Missing()
	f.Run(context.Background(), ids)

	g := load(t, f, 10)
	if !g.HasMeta || *g.ReviewPct != 94 || g.ReleaseDate != "2024-05-06" {
		t.Fatalf("meta not stored: %+v", g)
	}
	if len(g.Tags) != 2 || g.Tags[0] != "Roguelike" || len(g.Genres) != 2 || len(g.Categories) != 2 {
		t.Fatalf("tags not stored: %+v", g)
	}
	if g := load(t, f, 30); !g.Hidden || g.HiddenReason != "not a game" {
		t.Fatalf("non-game should be hidden: %+v", g)
	}

	fails, _ := f.Failures()
	if len(fails) != 1 || fails[0].AppID != 20 || fails[0].Reason != ErrNotFound.Error() {
		t.Fatalf("failures = %+v", fails)
	}
	if n, _ := f.MissingCount(); n != 1 {
		t.Fatalf("missing after = %d", n)
	}
	if f.Progress().Running {
		t.Fatal("progress should reset after the run")
	}
}

func TestMarkUnavailableStopsCountingAndFetching(t *testing.T) {
	f, _ := setup(t, &fakeSource{})
	if err := f.SetUnavailable(20, true); err != nil {
		t.Fatal(err)
	}
	ids, _ := f.Missing()
	for _, id := range ids {
		if id == 20 {
			t.Fatal("unavailable game must not be queued")
		}
	}
	if n, _ := f.MissingCount(); n != 2 {
		t.Fatalf("missing = %d", n)
	}
	if err := f.SetUnavailable(20, false); err != nil {
		t.Fatal(err)
	}
	if n, _ := f.MissingCount(); n != 3 {
		t.Fatalf("missing after undo = %d", n)
	}
}

func TestRetryClearsFailure(t *testing.T) {
	src := &fakeSource{items: map[int]Item{}}
	f, _ := setup(t, src)
	f.Run(context.Background(), []int{10})
	if fails, _ := f.Failures(); len(fails) != 1 {
		t.Fatalf("expected one failure, got %v", fails)
	}
	src.items[10] = Item{IsGame: true}
	retry, _ := f.Failed()
	f.Run(context.Background(), retry)
	if fails, _ := f.Failures(); len(fails) != 0 {
		t.Fatalf("failure not cleared: %v", fails)
	}
}

func TestPauseBlocksUntilResumed(t *testing.T) {
	src := &fakeSource{items: map[int]Item{10: {IsGame: true}}}
	f, _ := setup(t, src)
	f.progress.Running = true
	f.TogglePause()

	done := make(chan struct{})
	go func() { f.Run(context.Background(), []int{10}); close(done) }()
	select {
	case <-done:
		t.Fatal("run finished while paused")
	case <-time.After(100 * time.Millisecond):
	}
	f.TogglePause()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("run did not resume")
	}
	if src.calls != 1 {
		t.Fatalf("items calls = %d", src.calls)
	}
}
