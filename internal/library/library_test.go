package library

import (
	"database/sql"
	"testing"
	"time"

	"somanygames/internal/steam"
	"somanygames/internal/store"
)

var now = time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)

func daysAgo(d int) *time.Time { t := now.AddDate(0, 0, -d); return &t }

func newService(t *testing.T) (*Service, store.Settings) {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s := store.DefaultSettings()
	s.SteamID = "76561190000000000"
	if err := store.SaveSettings(db, s); err != nil {
		t.Fatal(err)
	}
	return &Service{DB: db, Now: func() time.Time { return now }}, s
}

func gameByID(t *testing.T, db *sql.DB, id int) Game {
	t.Helper()
	games, err := Load(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range games {
		if g.AppID == id {
			return g
		}
	}
	t.Fatalf("game %d not found", id)
	return Game{}
}

func TestAfterSync(t *testing.T) {
	cases := []struct {
		from       Status
		prev, next int
		want       Status
	}{
		{Unplayed, 0, 30, Playing},
		{ToPlay, 0, 5, Playing},
		{Shelved, 600, 660, Playing},
		{ToPlay, 0, 0, ToPlay}, // ADR 0002: 0 minutes never resets To play
		{Shelved, 600, 600, Shelved},
		{Finished, 100, 200, Finished},
		{NotForMe, 10, 20, NotForMe},
		{Endless, 10, 20, Endless},
		{Dead, 10, 20, Dead},
		{Playing, 10, 20, Playing},
	}
	for _, c := range cases {
		if got := AfterSync(c.from, c.prev, c.next); got != c.want {
			t.Errorf("AfterSync(%s, %d, %d) = %s, want %s", c.from, c.prev, c.next, got, c.want)
		}
	}
}

func TestClassifyDefault(t *testing.T) {
	if got := ClassifyDefault(daysAgo(10), now, 30); got != Playing {
		t.Errorf("played 10 days ago = %s", got)
	}
	if got := ClassifyDefault(daysAgo(31), now, 30); got != Shelved {
		t.Errorf("played 31 days ago = %s", got)
	}
	if got := ClassifyDefault(nil, now, 30); got != Shelved {
		t.Errorf("never played = %s", got)
	}
}

func TestHiddenReason(t *testing.T) {
	pats := store.DefaultSettings().HiddenPatterns
	notGame := false
	cases := map[string]string{
		"Paladins - Public Test":                 "Public Test",
		"Vaultbreakers Playtest":                 "Playtest",
		"Starbound - Unstable":                   "Unstable",
		"Demolish & Build":                       "",
		"Alphabet Soup":                          "",
		"Some Game Soundtrack":                   "Soundtrack",
		"Totally Reliable Delivery Service Beta": "Beta",
	}
	for name, want := range cases {
		if got := HiddenReason(name, pats, nil); got != want {
			t.Errorf("HiddenReason(%q) = %q, want %q", name, got, want)
		}
	}
	if got := HiddenReason("Plain Name", pats, &notGame); got != "not a game" {
		t.Errorf("non-game type = %q", got)
	}
}

func TestFirstImportThenClassify(t *testing.T) {
	svc, settings := newService(t)
	owned := []steam.Owned{
		{AppID: 1, Name: "Never Played", Minutes: 0},
		{AppID: 2, Name: "Recent", Minutes: 600, LastPlayed: daysAgo(3)},
		{AppID: 3, Name: "Old", Minutes: 1200, LastPlayed: daysAgo(400)},
		{AppID: 4, Name: "Paladins - Public Test", Minutes: 0},
	}
	res, err := svc.Import(owned, settings)
	if err != nil {
		t.Fatal(err)
	}
	if !res.FirstImport || res.NewGames != 4 {
		t.Fatalf("result = %+v", res)
	}

	if g := gameByID(t, svc.DB, 1); g.Status != Unplayed || !g.Classified {
		t.Errorf("0-minute game: %+v", g)
	}
	if g := gameByID(t, svc.DB, 2); g.Status != Playing || g.Classified {
		t.Errorf("recent game should be staged Playing and unclassified: %+v", g)
	}
	if g := gameByID(t, svc.DB, 3); g.Status != Shelved || g.Classified {
		t.Errorf("old game should be staged Shelved and unclassified: %+v", g)
	}
	if g := gameByID(t, svc.DB, 4); !g.Hidden || g.HiddenReason != "Public Test" {
		t.Errorf("test server should be hidden: %+v", g)
	}
	if n := unclassified(t, svc.DB); n != 2 {
		t.Fatalf("unclassified = %d", n)
	}

	if err := svc.SaveClassification(map[int]Status{2: Playing}); err == nil {
		t.Fatal("saving with a game missing should fail")
	}
	if err := svc.SaveClassification(map[int]Status{2: Playing, 3: Finished}); err != nil {
		t.Fatal(err)
	}
	if n := unclassified(t, svc.DB); n != 0 {
		t.Fatalf("unclassified after save = %d", n)
	}
	if g := gameByID(t, svc.DB, 3); g.Status != Finished || !g.Classified {
		t.Errorf("classified game: %+v", g)
	}
}

func TestLaterSyncAppliesL2AndSkipsClassify(t *testing.T) {
	svc, settings := newService(t)
	if _, err := svc.Import([]steam.Owned{
		{AppID: 1, Name: "A", Minutes: 0},
		{AppID: 2, Name: "B", Minutes: 0},
		{AppID: 3, Name: "C", Minutes: 0},
	}, settings); err != nil {
		t.Fatal(err)
	}
	settings, _ = store.LoadSettings(svc.DB)
	if _, err := svc.SetStatus([]int{2}, ToPlay); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetStatus([]int{3}, Finished); err != nil {
		t.Fatal(err)
	}

	res, err := svc.Import([]steam.Owned{
		{AppID: 1, Name: "A", Minutes: 30, LastPlayed: daysAgo(0)}, // Unplayed -> Playing
		{AppID: 2, Name: "B", Minutes: 0},                          // To play stays (ADR 0002)
		{AppID: 3, Name: "C", Minutes: 90, LastPlayed: daysAgo(0)}, // Finished never changes
		{AppID: 9, Name: "New", Minutes: 45, LastPlayed: daysAgo(1)},
	}, settings)
	if err != nil {
		t.Fatal(err)
	}
	if res.FirstImport || res.NewGames != 1 || res.AutoChanged != 1 {
		t.Fatalf("result = %+v", res)
	}
	want := map[int]Status{1: Playing, 2: ToPlay, 3: Finished, 9: Playing}
	for id, st := range want {
		if g := gameByID(t, svc.DB, id); g.Status != st || !g.Classified {
			t.Errorf("game %d = %s classified=%v, want %s", id, g.Status, g.Classified, st)
		}
	}
	if g := gameByID(t, svc.DB, 1); g.Minutes != 30 || g.LastPlayed == nil {
		t.Errorf("snapshot not applied: %+v", g)
	}
}

func TestSetStatusClearsRatingAndUndoRestoresIt(t *testing.T) {
	svc, settings := newService(t)
	if _, err := svc.Import([]steam.Owned{{AppID: 1, Name: "A", Minutes: 0}}, settings); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetStatus([]int{1}, Finished); err != nil {
		t.Fatal(err)
	}
	eight := 8
	if err := svc.SetRating(1, &eight); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetCompleted(1, true); err != nil {
		t.Fatal(err)
	}

	before, err := svc.SetStatus([]int{1}, Shelved)
	if err != nil {
		t.Fatal(err)
	}
	if g := gameByID(t, svc.DB, 1); g.Rating != nil || g.Completed {
		t.Fatalf("leaving Finished should clear rating and Completed: %+v", g)
	}
	if err := svc.Restore(before); err != nil {
		t.Fatal(err)
	}
	g := gameByID(t, svc.DB, 1)
	if g.Status != Finished || g.Rating == nil || *g.Rating != 8 || !g.Completed {
		t.Fatalf("undo did not restore: %+v", g)
	}

	if err := svc.SetRating(1, ptr(11)); err == nil {
		t.Error("rating 11 should be rejected")
	}
	if _, err := svc.SetStatus([]int{1}, Playing); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetRating(1, &eight); err == nil {
		t.Error("only Finished games take a rating")
	}
}

func TestHideOverrides(t *testing.T) {
	svc, settings := newService(t)
	if _, err := svc.Import([]steam.Owned{
		{AppID: 1, Name: "Thing Playtest", Minutes: 0},
		{AppID: 2, Name: "Real Game", Minutes: 0},
	}, settings); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetHideOverride(1, OverrideShow, settings.HiddenPatterns); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetHideOverride(2, OverrideHide, settings.HiddenPatterns); err != nil {
		t.Fatal(err)
	}
	if g := gameByID(t, svc.DB, 1); g.Hidden {
		t.Error("Not clutter should unhide")
	}
	if g := gameByID(t, svc.DB, 2); !g.Hidden || g.HiddenReason != "manual" {
		t.Errorf("manual hide: %+v", g)
	}
}

func ptr(n int) *int { return &n }

func unclassified(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM game WHERE classified = 0 AND hidden = 0`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestFinishedDateIsSetClearedAndRestored(t *testing.T) {
	svc, settings := newService(t)
	if _, err := svc.Import([]steam.Owned{{AppID: 1, Name: "A"}, {AppID: 2, Name: "B"}}, settings); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetStatus([]int{1}, Finished); err != nil {
		t.Fatal(err)
	}
	if g := gameByID(t, svc.DB, 1); g.FinishedAt == nil || *g.FinishedAt != "2026-11-07" {
		t.Fatalf("finishing should date it today: %v", g.FinishedAt)
	}
	if err := svc.SetFinishedAt(1, "2024-05-02"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"2026-11-08", "5/2/2024"} {
		if err := svc.SetFinishedAt(1, bad); err == nil {
			t.Errorf("date %q should be rejected", bad)
		}
	}
	if err := svc.SetFinishedAt(2, "2024-05-02"); err == nil {
		t.Error("only Finished games take a finished date")
	}

	before, err := svc.SetStatus([]int{1}, Shelved)
	if err != nil {
		t.Fatal(err)
	}
	if g := gameByID(t, svc.DB, 1); g.FinishedAt != nil {
		t.Fatalf("leaving Finished should clear the date: %v", *g.FinishedAt)
	}
	if err := svc.Restore(before); err != nil {
		t.Fatal(err)
	}
	if g := gameByID(t, svc.DB, 1); g.FinishedAt == nil || *g.FinishedAt != "2024-05-02" {
		t.Fatalf("undo should restore the backfilled date: %v", g.FinishedAt)
	}
	if err := svc.SetFinishedAt(1, ""); err != nil {
		t.Fatal(err)
	}
	if g := gameByID(t, svc.DB, 1); g.FinishedAt != nil {
		t.Fatal("an empty date means unknown")
	}
}

func TestTiersReplaceAndValidate(t *testing.T) {
	svc, settings := newService(t)
	if _, err := svc.Import([]steam.Owned{{AppID: 1, Name: "A"}, {AppID: 2, Name: "B"}}, settings); err != nil {
		t.Fatal(err)
	}
	if err := SaveTiers(svc.DB, []TierEntry{{1, "S", 0}, {2, "S", 1}, {99, "A", 0}}); err != nil {
		t.Fatal(err)
	}
	got, err := LoadTiers(svc.DB)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].AppID != 1 || got[1].AppID != 2 {
		t.Fatalf("tiers = %+v (unknown games are dropped)", got)
	}
	if err := SaveTiers(svc.DB, []TierEntry{{2, "F", 0}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := LoadTiers(svc.DB); len(got) != 1 || got[0].Tier != "F" {
		t.Fatalf("saving should replace the list: %+v", got)
	}
	if err := SaveTiers(svc.DB, []TierEntry{{1, "Z", 0}}); err == nil {
		t.Error("unknown tier should be rejected")
	}
	if got, _ := LoadTiers(svc.DB); len(got) != 1 {
		t.Fatal("a rejected save should leave the list as it was")
	}
}
