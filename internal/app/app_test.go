package app

import (
	"context"
	"testing"
	"time"

	"somanygames/internal/events"
	"somanygames/internal/library"
	"somanygames/internal/steam"
	"somanygames/internal/store"
)

type fakeOwned []steam.Owned

func (f fakeOwned) OwnedGames(context.Context, string) ([]steam.Owned, error) { return f, nil }

// newCore wires a Core over a temp database and a fake owned-games source.
func newCore(t *testing.T, owned fakeOwned) *Core {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	now := func() time.Time { return time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC) }
	c := &Core{DB: db, Now: now, Emit: func(string, any) {}, ctx: context.Background()}
	c.Library = &library.Service{DB: db, Source: owned, Now: c.now}
	s := store.DefaultSettings()
	s.SteamID = "1"
	if err := store.SaveSettings(db, s); err != nil {
		t.Fatal(err)
	}
	return c
}

func countEvents(t *testing.T, c *Core, kind string) int {
	t.Helper()
	var n int
	if err := c.DB.QueryRow(`SELECT COUNT(*) FROM event WHERE kind = ?`, kind).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSyncLogsSnapshotAndBacklog(t *testing.T) {
	c := newCore(t, fakeOwned{{AppID: 1, Name: "A"}, {AppID: 2, Name: "B", Minutes: 30}})
	if _, err := c.Sync(); err != nil {
		t.Fatal(err)
	}
	if countEvents(t, c, events.PlaytimeSnapshot) != 1 || countEvents(t, c, events.BacklogSnapshot) != 1 {
		t.Fatal("a sync should log one playtime and one backlog snapshot")
	}
	if c.SyncState().Running {
		t.Fatal("sync state should be idle afterwards")
	}
}

func TestRecommendationEventsAreLogged(t *testing.T) {
	c := newCore(t, nil)
	_, err := c.DB.Exec(`INSERT INTO game (appid, name, status, first_seen_at) VALUES
		(1, 'Played', 'shelved', 'x'), (2, 'Candidate', 'unplayed', 'x');
		INSERT INTO playtime_snapshot VALUES (1, 'x', 6000, NULL);
		INSERT INTO game_meta (appid, fetched_at, review_pct, review_count, is_game) VALUES (2, 'x', 90, 500, 1);
		INSERT INTO game_tag VALUES (1, 'tag', 'Roguelike', 100), (2, 'tag', 'Roguelike', 100);`)
	if err != nil {
		t.Fatal(err)
	}
	rec := &RecService{C: c}
	for i := 0; i < 2; i++ {
		r, err := rec.Rows("all")
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Rows) != 1 || r.Rows[0].Games[0].Game.AppID != 2 {
			t.Fatalf("rows = %+v", r.Rows)
		}
		if err := rec.Shown([]int{2}, "Roguelike"); err != nil {
			t.Fatal(err)
		}
	}
	if n := countEvents(t, c, events.RecShown); n != 1 {
		t.Fatalf("rec_shown logged %d times, want once per run", n)
	}
	if err := rec.Opened(2); err != nil {
		t.Fatal(err)
	}
	if err := rec.Dismissed(2); err != nil {
		t.Fatal(err)
	}
	if countEvents(t, c, events.RecOpened) != 1 || countEvents(t, c, events.RecDismissed) != 1 {
		t.Fatal("opened and dismissed should each be logged")
	}
}
