package meta

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"somanygames/internal/library"
	"somanygames/internal/store"
)

// Progress is pushed to the UI while a fetch runs (S5).
type Progress struct {
	Running bool   `json:"running"`
	Paused  bool   `json:"paused"`
	Current string `json:"current"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
}

// Failure is one game whose last fetch failed (S6).
type Failure struct {
	AppID  int    `json:"appid"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Fetcher runs one fetch at a time on a single worker goroutine.
type Fetcher struct {
	DB         *sql.DB
	Source     Source
	CoversDir  string
	Now        func() time.Time
	OnProgress func(Progress) // called on every change; may be nil
	OnFinish   func()         // called once a run ends; may be nil

	mu       sync.Mutex
	cond     *sync.Cond
	progress Progress
}

// ErrBusy means a fetch is already running.
var ErrBusy = errors.New("a fetch is already running")

// missingWhere selects visible games with no store data that are not marked unavailable.
const missingWhere = `WHERE g.hidden = 0 AND g.meta_unavailable = 0
	AND NOT EXISTS (SELECT 1 FROM game_meta m WHERE m.appid = g.appid)`

// MissingCount is the S4 banner number.
func (f *Fetcher) MissingCount() (int, error) {
	var n int
	err := f.DB.QueryRow(`SELECT COUNT(*) FROM game g ` + missingWhere).Scan(&n)
	return n, err
}

// Missing lists the games the Fetch button queues, in library order.
func (f *Fetcher) Missing() ([]int, error) {
	return f.ids(`SELECT g.appid FROM game g ` + missingWhere + ` ORDER BY g.name COLLATE NOCASE`)
}

// Failed lists the games in the failures list, for Retry all.
func (f *Fetcher) Failed() ([]int, error) {
	failures, err := f.Failures()
	ids := make([]int, len(failures))
	for i, fl := range failures {
		ids[i] = fl.AppID
	}
	return ids, err
}

// AllFetchable lists every game Re-fetch all refreshes (S7).
func (f *Fetcher) AllFetchable() ([]int, error) {
	return f.ids(`SELECT appid FROM game WHERE hidden = 0 AND meta_unavailable = 0 ORDER BY name COLLATE NOCASE`)
}

// Failures lists failed fetches with their reasons.
func (f *Fetcher) Failures() ([]Failure, error) {
	rows, err := f.DB.Query(`SELECT f.appid, g.name, f.reason FROM fetch_failure f JOIN game g ON g.appid = f.appid
		WHERE g.meta_unavailable = 0 ORDER BY g.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Failure{}
	for rows.Next() {
		var fl Failure
		if err := rows.Scan(&fl.AppID, &fl.Name, &fl.Reason); err != nil {
			return nil, err
		}
		out = append(out, fl)
	}
	return out, rows.Err()
}

// SetUnavailable marks or unmarks a game as having no store data to fetch (ADR 0003).
func (f *Fetcher) SetUnavailable(appid int, unavailable bool) error {
	_, err := f.DB.Exec(`UPDATE game SET meta_unavailable = ? WHERE appid = ?`, unavailable, appid)
	return err
}

// Progress returns the current state.
func (f *Fetcher) Progress() Progress {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.progress
}

// TogglePause pauses a running fetch or resumes a paused one (the single S5 button).
func (f *Fetcher) TogglePause() Progress {
	f.mu.Lock()
	if f.progress.Running {
		f.progress.Paused = !f.progress.Paused
		f.lazyCond().Broadcast()
	}
	p := f.progress
	f.mu.Unlock()
	f.emit(p)
	return p
}

// Start fetches appids in the background. Cancelling ctx stops the run.
func (f *Fetcher) Start(ctx context.Context, appids []int) error {
	f.mu.Lock()
	if f.progress.Running {
		f.mu.Unlock()
		return ErrBusy
	}
	f.progress = Progress{Running: len(appids) > 0, Total: len(appids)}
	f.mu.Unlock()
	if len(appids) == 0 {
		return nil
	}
	go func() {
		f.Run(ctx, appids)
		if f.OnFinish != nil {
			f.OnFinish()
		}
	}()
	return nil
}

// Run fetches appids on the calling goroutine; Start wraps it. Exported for tests.
func (f *Fetcher) Run(ctx context.Context, appids []int) {
	defer func() {
		f.mu.Lock()
		f.progress = Progress{}
		f.mu.Unlock()
		f.emit(Progress{})
	}()
	f.set(func(p *Progress) { p.Running, p.Total = true, len(appids) })
	names := f.names(appids)

	for start := 0; start < len(appids); start += BatchSize {
		batch := appids[start:min(start+BatchSize, len(appids))]
		if !f.waitWhilePaused(ctx) {
			return
		}
		items, err := f.Source.Items(ctx, batch)
		if ctx.Err() != nil {
			return
		}
		for _, id := range batch {
			if !f.waitWhilePaused(ctx) {
				return
			}
			f.set(func(p *Progress) { p.Current = names[id] })
			switch item, ok := items[id]; {
			case err != nil:
				f.fail(id, err)
			case !ok:
				f.fail(id, ErrNotFound)
			default:
				if err := f.save(ctx, id, item); err != nil {
					f.fail(id, err)
				}
			}
			if ctx.Err() != nil {
				return
			}
			f.set(func(p *Progress) { p.Done++ })
		}
		f.recomputeHidden()
	}
}

// save stores one game's data. A missing genre list or cover is not a failure (S9).
func (f *Fetcher) save(ctx context.Context, appid int, it Item) error {
	genres, _ := f.Source.Genres(ctx, appid)
	cover := ""
	if it.CoverURL != "" {
		if img, err := f.Source.Cover(ctx, it.CoverURL); err == nil && len(img) > 0 {
			name := strconv.Itoa(appid) + ".jpg"
			if err := os.MkdirAll(f.CoversDir, 0o755); err == nil && os.WriteFile(filepath.Join(f.CoversDir, name), img, 0o644) == nil {
				cover = name
			}
		}
	}

	tx, err := f.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT OR REPLACE INTO game_meta (appid, fetched_at, release_date, review_pct, review_count, is_game, cover_path)
		VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, ?)`,
		appid, f.Now().UTC().Format(store.TimeFormat), it.ReleaseDate, it.ReviewPct, it.ReviewCount, it.IsGame, cover); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM game_tag WHERE appid = ?`, appid); err != nil {
		return err
	}
	insert := func(kind, name string, weight any) error {
		_, err := tx.Exec(`INSERT OR IGNORE INTO game_tag (appid, kind, name, weight) VALUES (?, ?, ?, ?)`, appid, kind, name, weight)
		return err
	}
	for _, t := range it.Tags {
		if err := insert("tag", t.Name, t.Weight); err != nil {
			return err
		}
	}
	for _, g := range genres {
		if err := insert("genre", g, nil); err != nil {
			return err
		}
	}
	for _, c := range it.Categories {
		if err := insert("category", c, nil); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM fetch_failure WHERE appid = ?`, appid); err != nil {
		return err
	}
	return tx.Commit()
}

func (f *Fetcher) fail(appid int, cause error) {
	_, _ = f.DB.Exec(`INSERT OR REPLACE INTO fetch_failure (appid, reason, at) VALUES (?, ?, ?)`,
		appid, cause.Error(), f.Now().UTC().Format(store.TimeFormat))
}

// recomputeHidden applies the store type to the hidden flags (L6) after each batch.
func (f *Fetcher) recomputeHidden() {
	settings, err := store.LoadSettings(f.DB)
	if err != nil {
		return
	}
	tx, err := f.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	if library.RecomputeHidden(tx, settings.HiddenPatterns) == nil {
		_ = tx.Commit()
	}
}

func (f *Fetcher) names(appids []int) map[int]string {
	out := map[int]string{}
	for _, id := range appids {
		var n string
		if f.DB.QueryRow(`SELECT name FROM game WHERE appid = ?`, id).Scan(&n) == nil {
			out[id] = n
		}
	}
	return out
}

func (f *Fetcher) ids(query string) ([]int, error) {
	rows, err := f.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// waitWhilePaused blocks while paused; false means the run was cancelled.
func (f *Fetcher) waitWhilePaused(ctx context.Context) bool {
	stop := context.AfterFunc(ctx, func() {
		f.mu.Lock()
		f.lazyCond().Broadcast()
		f.mu.Unlock()
	})
	defer stop()
	f.mu.Lock()
	defer f.mu.Unlock()
	for f.progress.Paused && ctx.Err() == nil {
		f.lazyCond().Wait()
	}
	return ctx.Err() == nil
}

func (f *Fetcher) set(change func(*Progress)) {
	f.mu.Lock()
	change(&f.progress)
	p := f.progress
	f.mu.Unlock()
	f.emit(p)
}

func (f *Fetcher) emit(p Progress) {
	if f.OnProgress != nil {
		f.OnProgress(p)
	}
}

// lazyCond must be called with f.mu held.
func (f *Fetcher) lazyCond() *sync.Cond {
	if f.cond == nil {
		f.cond = sync.NewCond(&f.mu)
	}
	return f.cond
}
