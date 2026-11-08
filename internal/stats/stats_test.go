package stats

import (
	"testing"
	"time"

	"somanygames/internal/library"
	"somanygames/internal/steam"
	"somanygames/internal/store"
)

func TestComputeCountsAndLatestFinished(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	clock := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	lib := &library.Service{DB: db, Now: func() time.Time { return clock }}
	settings := store.DefaultSettings()
	last := clock.AddDate(-1, 0, 0)
	if _, err := lib.Import([]steam.Owned{
		{AppID: 1, Name: "Classified Finished", Minutes: 600, LastPlayed: &last},
		{AppID: 2, Name: "Finished Later", Minutes: 0},
		{AppID: 3, Name: "Hidden Demo", Minutes: 0},
	}, settings); err != nil {
		t.Fatal(err)
	}
	if err := lib.SaveClassification(map[int]library.Status{1: library.Finished}); err != nil {
		t.Fatal(err)
	}
	if _, err := lib.SetStatus([]int{2}, library.Finished); err != nil {
		t.Fatal(err)
	}

	games, _ := library.Load(db)
	s := Compute(games, settings, clock)
	total, finished := 0.0, 0.0
	for _, b := range s.StatusCounts {
		total += b.Value
		if b.Label == string(library.Finished) {
			finished = b.Value
		}
	}
	if total != 2 || finished != 2 {
		t.Fatalf("hidden game should not count: %+v", s.StatusCounts)
	}
	// The classified game has no known date, so the one finished in the app is latest.
	if s.LatestFinished == nil || s.LatestFinished.AppID != 2 || *s.LatestFinished.FinishedAt != "2026-03-01" {
		t.Fatalf("latest finished = %+v", s.LatestFinished)
	}
	if len(s.MostPlayed) != 1 || s.MostPlayed[0].AppID != 1 {
		t.Fatalf("most played = %+v", s.MostPlayed)
	}
}
