package store

import (
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
	if s.HoursLimit != 10 || s.MinReviews != 50 || s.AbandonedDays != 60 || len(s.HiddenPatterns) != 9 {
		t.Fatalf("defaults wrong: %+v", s)
	}

	s.HoursLimit = 15
	s.Pinned = []string{"Roguelike"}
	if err := SaveSettings(db, s); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSettings(db)
	if err != nil {
		t.Fatal(err)
	}
	if got.HoursLimit != 15 || len(got.Pinned) != 1 || got.Pinned[0] != "Roguelike" {
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
