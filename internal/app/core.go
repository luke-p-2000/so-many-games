// Package app holds the thin Wails services the frontend calls. Each service only wires a
// request to the package that owns the rule; no business logic lives here.
package app

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"time"

	"somanygames/internal/library"
	"somanygames/internal/meta"
	"somanygames/internal/steam"
	"somanygames/internal/store"
)

// Frontend event names.
const (
	EventLibraryChanged = "library:changed" // payload: none; refetch games
	EventSync           = "sync:state"      // payload: SyncState
	EventFetchProgress  = "fetch:progress"  // payload: meta.Progress
)

// Core is the shared state behind every service.
type Core struct {
	DB      *sql.DB
	DataDir string
	Keys    steam.KeyStore
	Steam   *steam.WebAPI
	Library *library.Service
	Fetcher *meta.Fetcher
	Emit    func(name string, data any)
	Now     func() time.Time
	Version string

	ctx     context.Context
	syncMu  sync.Mutex
	syncing SyncState
}

// SyncState is shown in Settings and while the launch sync runs.
type SyncState struct {
	Running bool   `json:"running"`
	Error   string `json:"error"`
}

// NewCore opens the database in dataDir and wires the services' dependencies.
func NewCore(dataDir string) (*Core, error) {
	db, err := store.Open(dataDir)
	if err != nil {
		return nil, err
	}
	httpc := steam.NewHTTP()
	keys := steam.CredentialManager{}
	api := &steam.WebAPI{HTTP: httpc, Keys: keys}
	c := &Core{
		DB:      db,
		DataDir: dataDir,
		Keys:    keys,
		Steam:   api,
		Now:     time.Now,
		Emit:    func(string, any) {},
		ctx:     context.Background(),
	}
	c.Library = &library.Service{DB: db, Source: api, Now: c.now}
	c.Fetcher = &meta.Fetcher{
		DB:         db,
		Source:     &meta.SteamStore{HTTP: httpc},
		CoversDir:  c.CoversDir(),
		Now:        c.now,
		OnProgress: func(p meta.Progress) { c.Emit(EventFetchProgress, p) },
		OnFinish:   func() { c.Emit(EventLibraryChanged, nil) },
	}
	return c, nil
}

func (c *Core) now() time.Time { return c.Now() }

// CoversDir is where cover art is cached beside the database.
func (c *Core) CoversDir() string { return filepath.Join(c.DataDir, "covers") }

// Start records the app context and runs the launch sync (S3) when Steam is connected.
func (c *Core) Start(ctx context.Context) {
	c.ctx = ctx
	if s, err := store.LoadSettings(c.DB); err == nil && s.SteamID != "" {
		go c.Sync()
	}
}

// Sync runs one library sync. Errors are reported
// through the sync state, never blocking the UI (non-functional: UI never blocks on network).
func (c *Core) Sync() (library.SyncResult, error) {
	c.syncMu.Lock()
	if c.syncing.Running {
		c.syncMu.Unlock()
		return library.SyncResult{}, errors.New("a sync is already running")
	}
	c.syncing = SyncState{Running: true}
	c.syncMu.Unlock()
	c.Emit(EventSync, SyncState{Running: true})

	res, err := c.Library.Sync(c.ctx)
	state := SyncState{}
	if err != nil {
		state.Error = err.Error()
	}
	c.syncMu.Lock()
	c.syncing = state
	c.syncMu.Unlock()
	c.Emit(EventSync, state)
	c.Emit(EventLibraryChanged, nil)
	return res, err
}

// SyncState returns the current sync state.
func (c *Core) SyncState() SyncState {
	c.syncMu.Lock()
	defer c.syncMu.Unlock()
	return c.syncing
}

// Settings loads the current settings.
func (c *Core) Settings() (store.Settings, error) { return store.LoadSettings(c.DB) }

// gamesAndSettings loads what most rules need: every game and the current settings.
func (c *Core) gamesAndSettings() ([]library.Game, store.Settings, error) {
	games, err := library.Load(c.DB)
	if err != nil {
		return nil, store.Settings{}, err
	}
	s, err := c.Settings()
	return games, s, err
}

// Context is the app lifetime context.
func (c *Core) Context() context.Context { return c.ctx }
