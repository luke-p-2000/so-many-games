// Package store owns the SQLite database: opening, migrations and file backups (B1).
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

const (
	dbFile          = "somanygames.db"
	backupDir       = "backups"
	backupsToKeep   = 3
	timestampLayout = "20060102-150405"
)

// TimeFormat is how every timestamp is stored: UTC RFC 3339, which sorts as text.
const TimeFormat = time.RFC3339

// Open opens (creating if needed) the database in dir and applies pending migrations,
// copying the existing database into dir/backups first and keeping the newest three copies.
func Open(dir string) (*sql.DB, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, dbFile)
	_, statErr := os.Stat(path)
	existed := statErr == nil

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	// One connection serialises access; ample for one user and avoids SQLITE_BUSY between goroutines.
	db.SetMaxOpenConns(1)

	pending, err := pendingMigrations(db)
	if err != nil {
		db.Close()
		return nil, err
	}
	if existed && len(pending) > 0 {
		if err := backupBeforeUpgrade(db, dir); err != nil {
			db.Close()
			return nil, fmt.Errorf("pre-upgrade backup: %w", err)
		}
	}
	if err := applyMigrations(db, pending); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Snapshot writes a consistent copy of the database to dest (used for B1 backups).
func Snapshot(db *sql.DB, dest string) error {
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	_, err := db.Exec(`VACUUM INTO ?`, dest)
	return err
}

type migration struct {
	version int
	name    string
}

func pendingMigrations(db *sql.DB) ([]migration, error) {
	var current int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&current); err != nil {
		return nil, err
	}
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	var out []migration
	for _, e := range entries {
		v, err := strconv.Atoi(strings.SplitN(e.Name(), "_", 2)[0])
		if err != nil {
			return nil, fmt.Errorf("migration %s: name must start with a number", e.Name())
		}
		if v > current {
			out = append(out, migration{v, e.Name()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func applyMigrations(db *sql.DB, pending []migration) error {
	for _, m := range pending {
		body, err := migrationFS.ReadFile("migrations/" + m.name)
		if err != nil {
			return err
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		// PRAGMA cannot take a bound parameter; the version is an integer we parsed.
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, m.version)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func backupBeforeUpgrade(db *sql.DB, dir string) error {
	bdir := filepath.Join(dir, backupDir)
	if err := os.MkdirAll(bdir, 0o755); err != nil {
		return err
	}
	name := "pre-upgrade-" + time.Now().Format(timestampLayout) + ".db"
	if err := Snapshot(db, filepath.Join(bdir, name)); err != nil {
		return err
	}
	return pruneBackups(bdir, backupsToKeep)
}

// pruneBackups keeps the newest n pre-upgrade copies; names sort by timestamp.
func pruneBackups(bdir string, n int) error {
	matches, err := filepath.Glob(filepath.Join(bdir, "pre-upgrade-*.db"))
	if err != nil {
		return err
	}
	sort.Strings(matches)
	for len(matches) > n {
		if err := os.Remove(matches[0]); err != nil {
			return err
		}
		matches = matches[1:]
	}
	return nil
}
