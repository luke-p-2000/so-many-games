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
	nine := 9
	if err := src.SetRating(1, &nine); err != nil {
		t.Fatal(err)
	}
	if err := library.SaveTiers(src.DB, []library.TierEntry{{AppID: 1, Tier: "A", Position: 0}}); err != nil {
		t.Fatal(err)
	}
	s, _ := store.LoadSettings(src.DB)
	s.Pinned = []string{"Roguelike"}
	if err := store.SaveSettings(src.DB, s); err != nil {
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
	if games[0].AppID != 1 || games[0].Status != library.Finished || games[0].Rating == nil || *games[0].Rating != 9 {
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
	} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func TestBackupWritesFile(t *testing.T) {
	lib := seeded(t)
	dest := filepath.Join(t.TempDir(), "backup.db")
	if err := Backup(lib.DB, dest); err != nil {
		t.Fatal(err)
	}
}
