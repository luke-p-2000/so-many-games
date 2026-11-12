package transfer

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"somanygames/internal/library"
	"somanygames/internal/steam"
	"somanygames/internal/store"
)

var now = time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)

func seeded(t *testing.T) *library.Service {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	lib := &library.Service{DB: db, Now: func() time.Time { return now }}
	if _, err := lib.Import([]steam.Owned{{AppID: 1, Name: "One"}, {AppID: 2, Name: "Two"}}, store.DefaultSettings()); err != nil {
		t.Fatal(err)
	}
	return lib
}

func TestExportImportRoundTrip(t *testing.T) {
	src := seeded(t)
	if _, err := src.SetStatus([]int{1}, library.Finished); err != nil {
		t.Fatal(err)
	}
	s := "S"
	if err := src.SetRating(1, &s); err != nil {
		t.Fatal(err)
	}
	if err := library.SaveTiers(src.DB, []library.TierEntry{{AppID: 1, Tier: "A", Position: 0}}); err != nil {
		t.Fatal(err)
	}
	set, _ := store.LoadSettings(src.DB)
	set.Pinned = []string{"Roguelike"}
	if err := store.SaveSettings(src.DB, set); err != nil {
		t.Fatal(err)
	}

	f, err := Export(src.DB, now)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(f)
	if strings.Contains(strings.ToLower(string(data)), "api_key") || strings.Contains(string(data), "steam_id") {
		t.Fatal("export must not carry the key or account settings")
	}

	dst := seeded(t)
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := Summarise(dst.DB, parsed)
	if err != nil {
		t.Fatal(err)
	}
	if sum.StatusChanges != 1 || sum.RatingChanges != 1 || !sum.TagSettingsDiff || sum.EventsInFile != len(f.Events) {
		t.Fatalf("summary = %+v", sum)
	}
	if err := Apply(dst.DB, parsed, now); err != nil {
		t.Fatal(err)
	}
	games, _ := library.Load(dst.DB)
	if games[0].AppID != 1 || games[0].Status != library.Finished || games[0].Rating == nil || *games[0].Rating != "S" {
		t.Fatalf("imported game = %+v", games[0])
	}
	if games[0].FinishedAt == nil || *games[0].FinishedAt != "2026-11-07" {
		t.Fatalf("finished date not imported: %v", games[0].FinishedAt)
	}
	if tiers, _ := library.LoadTiers(dst.DB); len(tiers) != 1 || tiers[0].Tier != "A" {
		t.Fatalf("tiers not imported: %+v", tiers)
	}
	got, _ := store.LoadSettings(dst.DB)
	if len(got.Pinned) != 1 {
		t.Fatalf("tag settings not imported: %+v", got.Pinned)
	}
	again, _ := Summarise(dst.DB, parsed)
	if again.StatusChanges != 0 || again.RatingChanges != 0 || again.TagSettingsDiff {
		t.Fatalf("second summary should be empty: %+v", again)
	}
}

func TestParseRejectsBadFiles(t *testing.T) {
	for _, bad := range []string{
		`not json`,
		`{"version": 99}`,
		`{"version": 1, "games": [{"appid": 1, "status": "paused"}]}`,
		`{"version": 1, "games": [{"appid": 1, "status": "finished", "rating": 11}]}`,
		`{"version": 1, "games": [{"appid": 1, "status": "finished", "rating": "S"}]}`,
		`{"version": 2, "games": [{"appid": 1, "status": "finished", "rating": 9}]}`,
		`{"version": 2, "games": [{"appid": 1, "status": "finished", "rating": "E"}]}`,
	} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func TestParseUpgradesVersion1Ratings(t *testing.T) {
	data := `{"version": 1, "games": [
		{"appid": 1, "status": "finished", "rating": 0},
		{"appid": 2, "status": "finished", "rating": 2},
		{"appid": 3, "status": "finished", "rating": 4},
		{"appid": 4, "status": "finished", "rating": 5},
		{"appid": 5, "status": "finished", "rating": 8},
		{"appid": 6, "status": "finished", "rating": 9},
		{"appid": 7, "status": "finished"}],
		"events": [{"at": "2026-11-07T00:00:00Z", "kind": "rated", "appid": 6, "payload": {"rating": 10}}]}`
	f, err := Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"F", "D", "C", "B", "A", "S"}
	for i, w := range want {
		if r := f.Games[i].Rating; r == nil || *r != w {
			t.Errorf("game %d rating = %v, want %s", i+1, r, w)
		}
	}
	if f.Games[6].Rating != nil {
		t.Errorf("unrated game got %v", *f.Games[6].Rating)
	}
	if string(f.Events[0].Payload) != `{"rating":"S"}` {
		t.Errorf("event payload = %s", f.Events[0].Payload)
	}
}

func TestBackupWritesFile(t *testing.T) {
	lib := seeded(t)
	dest := filepath.Join(t.TempDir(), "backup.db")
	if err := Backup(lib.DB, dest); err != nil {
		t.Fatal(err)
	}
}
