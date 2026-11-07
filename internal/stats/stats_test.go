package stats

import (
	"testing"
	"time"

	"somanygames/internal/library"
	"somanygames/internal/steam"
	"somanygames/internal/store"
)

func TestFinishedPerYearAndBurnDown(t *testing.T) {
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
	if err := LogBacklog(db, games, settings, clock); err != nil {
		t.Fatal(err)
	}
	s, err := Compute(db, games, settings, clock)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.FinishedPerYear) != 2 || s.FinishedPerYear[0].Label != BeforeFirstSync || s.FinishedPerYear[1].Label != "2026" {
		t.Fatalf("finished per year = %+v", s.FinishedPerYear)
	}
	total := 0.0
	for _, b := range s.StatusCounts {
		total += b.Value
	}
	if total != 2 {
		t.Fatalf("hidden game should not count: %+v", s.StatusCounts)
	}
	if len(s.BurnDown) != 1 || s.BurnDown[0].Value != 0 {
		t.Fatalf("burn-down = %+v", s.BurnDown)
	}
}
