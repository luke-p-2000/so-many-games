package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAppliesMigrationsOnce(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	var v int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil || v < 1 {
		t.Fatalf("user_version = %d, %v", v, err)
	}
	db.Close()

	// Reopening with nothing pending must not create a backup.
	db, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	if _, err := os.Stat(filepath.Join(dir, backupDir)); !os.IsNotExist(err) {
		t.Fatalf("unexpected backup dir: %v", err)
	}
}

func TestPruneBackupsKeepsNewestThree(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"20260101-000000", "20260102-000000", "20260103-000000", "20260104-000000"} {
		if err := os.WriteFile(filepath.Join(dir, "pre-upgrade-"+n+".db"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := pruneBackups(dir, 3); err != nil {
		t.Fatal(err)
	}
	left, _ := filepath.Glob(filepath.Join(dir, "*.db"))
	if len(left) != 3 || filepath.Base(left[0]) != "pre-upgrade-20260102-000000.db" {
		t.Fatalf("left = %v", left)
	}
}

func TestSettingsDefaultsAndRoundTrip(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s, err := LoadSettings(db)
	if err != nil {
		t.Fatal(err)
	}
	if s.ShelvedHours != 10 || s.EndlessHours != 10 || s.MinReviews != 50 || s.AbandonedDays != 60 || len(s.HiddenPatterns) != 9 {
		t.Fatalf("defaults wrong: %+v", s)
	}

	s.ShelvedHours, s.EndlessHours = 15, 4
	s.Pinned = []string{"Roguelike"}
	if err := SaveSettings(db, s); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSettings(db)
	if err != nil {
		t.Fatal(err)
	}
	if got.ShelvedHours != 15 || got.EndlessHours != 4 || len(got.Pinned) != 1 || got.Pinned[0] != "Roguelike" {
		t.Fatalf("round trip lost data: %+v", got)
	}

	s.WheelSize = 13
	if err := SaveSettings(db, s); err == nil {
		t.Fatal("expected invalid wheel size to be rejected")
	}
}

func TestSnapshotWritesCopy(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dest := filepath.Join(dir, "copy.db")
	if err := Snapshot(db, dest); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(dest); err != nil || fi.Size() == 0 {
		t.Fatalf("snapshot missing: %v", err)
	}
}

func TestLegacyHoursLimitCarriesOver(t *testing.T) {
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO setting (key, value) VALUES ('hours_limit', '7')`); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(db)
	if err != nil {
		t.Fatal(err)
	}
	if s.ShelvedHours != 7 || s.EndlessHours != 7 {
		t.Fatalf("old limit not carried over: %+v", s)
	}
	if err := SaveSettings(db, s); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM setting WHERE key = 'hours_limit'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("legacy key left behind: n=%d err=%v", n, err)
	}
}

func TestFinishedDateBackfill(t *testing.T) {
	dir := t.TempDir()
	// Build a version 1 database by hand, as an older install would have it.
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(dir, dbFile)))
	if err != nil {
		t.Fatal(err)
	}
	v1, _ := migrationFS.ReadFile("migrations/001_init.sql")
	if _, err := db.Exec(string(v1) + `;
		PRAGMA user_version = 1;
		INSERT INTO game (appid, name, status, first_seen_at) VALUES
			(1, 'Finished in app', 'finished', 'x'), (2, 'Classified', 'finished', 'x'), (3, 'Left', 'shelved', 'x');
		INSERT INTO event (at, kind, appid, payload) VALUES
			('2025-01-01T10:00:00Z', 'status_changed', 1, '{"to":"finished","source":"user"}'),
			('2025-06-01T10:00:00Z', 'status_changed', 1, '{"to":"finished","source":"user"}'),
			('2025-01-01T10:00:00Z', 'status_changed', 2, '{"to":"finished","source":"classify"}'),
			('2025-01-01T10:00:00Z', 'status_changed', 3, '{"to":"finished","source":"user"}');`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	db, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	want := map[int]string{1: "2025-06-01", 2: "", 3: ""}
	for id, w := range want {
		var got sql.NullString
		if err := db.QueryRow(`SELECT finished_at FROM game WHERE appid = ?`, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got.String != w {
			t.Errorf("game %d finished_at = %q, want %q", id, got.String, w)
		}
	}
}
