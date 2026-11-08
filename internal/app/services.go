package app

import (
	"errors"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"somanygames/internal/events"
	"somanygames/internal/library"
	"somanygames/internal/meta"
	"somanygames/internal/recommend"
	"somanygames/internal/stats"
	"somanygames/internal/store"
	"somanygames/internal/wheel"
)

// ---- Setup (S1, S2) ----

// SetupService connects Steam on first run and changes the key or profile later.
type SetupService struct{ C *Core }

// Connection describes what is saved, never the key itself.
type Connection struct {
	Connected  bool   `json:"connected"`
	HasKey     bool   `json:"hasKey"`
	ProfileURL string `json:"profileUrl"`
}

func (s *SetupService) Status() (Connection, error) {
	settings, err := s.C.Settings()
	if err != nil {
		return Connection{}, err
	}
	_, keyErr := s.C.Keys.Get()
	hasKey := keyErr == nil
	return Connection{Connected: hasKey && settings.SteamID != "", HasKey: hasKey, ProfileURL: settings.ProfileURL}, nil
}

// Version is the app version shown in the window corner.
func (s *SetupService) Version() string { return s.C.Version }

// Connect tests the key and profile, then saves them and runs the first sync.
func (s *SetupService) Connect(key, profile string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		if k, err := s.C.Keys.Get(); err == nil {
			key = k // changing only the profile keeps the saved key
		}
	}
	if len(key) != 32 {
		return errors.New("a Steam Web API key is 32 characters")
	}
	steamID, err := s.C.Steam.ResolveSteamID(s.C.Context(), key, profile)
	if err != nil {
		return err
	}
	if err := s.C.Steam.Check(s.C.Context(), key, steamID); err != nil {
		return err
	}
	if err := s.C.Keys.Set(key); err != nil {
		return err
	}
	settings, err := s.C.Settings()
	if err != nil {
		return err
	}
	settings.SteamID, settings.ProfileURL = steamID, strings.TrimSpace(profile)
	if err := store.SaveSettings(s.C.DB, settings); err != nil {
		return err
	}
	go s.C.Sync()
	return nil
}

// ---- Library (L1–L8, S3) ----

// LibraryService reads games and changes statuses.
type LibraryService struct{ C *Core }

func (s *LibraryService) Games() ([]library.Game, error) { return library.Load(s.C.DB) }

func (s *LibraryService) SyncNow() error {
	_, err := s.C.Sync()
	return err
}

func (s *LibraryService) SyncState() SyncState { return s.C.SyncState() }

func (s *LibraryService) SetStatus(appids []int, status string) ([]library.State, error) {
	st, err := library.ParseStatus(status)
	if err != nil {
		return nil, err
	}
	return s.C.Library.SetStatus(appids, st)
}

func (s *LibraryService) Restore(states []library.State) error { return s.C.Library.Restore(states) }

func (s *LibraryService) SetRating(appid int, rating *int) error {
	return s.C.Library.SetRating(appid, rating)
}

func (s *LibraryService) SetCompleted(appid int, completed bool) error {
	return s.C.Library.SetCompleted(appid, completed)
}

// Staged is one Classify-screen choice.
type Staged struct {
	AppID  int    `json:"appid"`
	Status string `json:"status"`
}

func (s *LibraryService) SaveClassification(staged []Staged) error {
	m := map[int]library.Status{}
	for _, st := range staged {
		v, err := library.ParseStatus(st.Status)
		if err != nil {
			return err
		}
		m[st.AppID] = v
	}
	if err := s.C.Library.SaveClassification(m); err != nil {
		return err
	}
	s.C.Emit(EventLibraryChanged, nil)
	return nil
}

// SetHideOverride applies Not clutter ("show"), Hide ("hide") or automatic ("").
func (s *LibraryService) SetHideOverride(appid int, override string) error {
	settings, err := s.C.Settings()
	if err != nil {
		return err
	}
	return s.C.Library.SetHideOverride(appid, override, settings.HiddenPatterns)
}

// ---- Store data (S4–S7) ----

// MetaService drives the fetch banner.
type MetaService struct{ C *Core }

// Banner is the S4/S5/S6 state.
type Banner struct {
	Missing  int            `json:"missing"`
	Progress meta.Progress  `json:"progress"`
	Failures []meta.Failure `json:"failures"`
}

func (s *MetaService) Banner() (Banner, error) {
	n, err := s.C.Fetcher.MissingCount()
	if err != nil {
		return Banner{}, err
	}
	f, err := s.C.Fetcher.Failures()
	return Banner{Missing: n, Progress: s.C.Fetcher.Progress(), Failures: f}, err
}

func (s *MetaService) Fetch() error      { return s.start(s.C.Fetcher.Missing) }
func (s *MetaService) RetryAll() error   { return s.start(s.C.Fetcher.Failed) }
func (s *MetaService) RefetchAll() error { return s.start(s.C.Fetcher.AllFetchable) }

// Refetch fetches the given games again (Retry and Re-fetch selected).
func (s *MetaService) Refetch(appids []int) error {
	return s.C.Fetcher.Start(s.C.Context(), appids)
}

func (s *MetaService) TogglePause() meta.Progress { return s.C.Fetcher.TogglePause() }

func (s *MetaService) SetUnavailable(appid int, unavailable bool) error {
	if err := s.C.Fetcher.SetUnavailable(appid, unavailable); err != nil {
		return err
	}
	s.C.Emit(EventLibraryChanged, nil)
	return nil
}

func (s *MetaService) start(list func() ([]int, error)) error {
	ids, err := list()
	if err != nil {
		return err
	}
	return s.C.Fetcher.Start(s.C.Context(), ids)
}

// ---- Recommendations (R1–R7) ----

// RecService serves recommendation rows and Pick back up.
type RecService struct {
	C     *Core
	mu    sync.Mutex
	shown map[int]bool // rec_shown is logged once per game per run
}

// Recommendations is one view of the rows.
type Recommendations struct {
	Profile  recommend.Profile  `json:"profile"`
	Tailored []recommend.Ranked `json:"tailored"` // games matching the most taste tags, shown first
	Rows     []recommend.Row    `json:"rows"`
}

// Rows returns the Tailored for you row and the "Because you play ..." rows for the
// All-time or Recent ("recent") profile.
func (s *RecService) Rows(view string) (Recommendations, error) {
	games, settings, err := s.C.gamesAndSettings()
	if err != nil {
		return Recommendations{}, err
	}
	p := s.profile(games, settings, view)
	ranked := recommend.TagEngine{MinReviews: settings.MinReviews}.Recommend(p, recommend.Pool(games, limits(settings)))
	return Recommendations{Profile: p, Tailored: recommend.Tailored(ranked), Rows: recommend.Rows(p, ranked)}, nil
}

func (s *RecService) PickBackUp() ([]recommend.BackUp, error) {
	games, settings, err := s.C.gamesAndSettings()
	if err != nil {
		return nil, err
	}
	return recommend.PickBackUp(games, s.profile(games, settings, "all")), nil
}

// Opened logs that a recommendation was opened; the frontend opens the store page.
func (s *RecService) Opened(appid int) error {
	return events.Log(s.C.DB, s.C.now(), events.RecOpened, appid, struct{}{})
}

func (s *RecService) Dismissed(appid int) error {
	return events.Log(s.C.DB, s.C.now(), events.RecDismissed, appid, struct{}{})
}

func (s *RecService) profile(games []library.Game, st store.Settings, view string) recommend.Profile {
	var since *time.Time
	if view == "recent" {
		t := recommend.RecentSince(s.C.now(), st.RecentMonths)
		since = &t
	}
	return recommend.BuildProfile(games, st.Stoplist, st.Pinned, st.Avoided, since)
}

// Shown logs the recommendations a section is showing, once per game per run (E1).
func (s *RecService) Shown(appids []int, section string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shown == nil {
		s.shown = map[int]bool{}
	}
	for _, id := range appids {
		if s.shown[id] {
			continue
		}
		s.shown[id] = true
		if err := events.Log(s.C.DB, s.C.now(), events.RecShown, id, map[string]string{"row": section}); err != nil {
			return err
		}
	}
	return nil
}

// ---- Wheel (W1–W3) ----

// WheelService fills the wheel from the filtered pool and picks the winner.
type WheelService struct{ C *Core }

// Wheel is the drawn wheel and how many pool games passed the filters.
type Wheel struct {
	Entries  []library.Game `json:"entries"`
	PoolSize int            `json:"poolSize"`
}

// Size returns the remembered wheel size (12, 15, 18, 24, or 0 for All).
func (s *WheelService) Size() (int, error) {
	settings, err := s.C.Settings()
	return settings.WheelSize, err
}

// SetSize remembers the size chosen with the wheel's slider (W2).
func (s *WheelService) SetSize(size int) error {
	settings, err := s.C.Settings()
	if err != nil {
		return err
	}
	settings.WheelSize = size
	return store.SaveSettings(s.C.DB, settings)
}

// Draw fills a wheel of the given size from the filtered pool, so the wheel can show its
// games before it spins.
func (s *WheelService) Draw(f wheel.Filters, bias bool, size int) (Wheel, error) {
	pool, weights, err := s.pool(f, bias)
	if err != nil {
		return Wheel{}, err
	}
	return Wheel{Entries: wheel.Draw(pool, size, weights, newRand()), PoolSize: len(pool)}, nil
}

// Pick spins the drawn wheel: it returns the winning index into appids, or -1 when none of
// them is still eligible.
func (s *WheelService) Pick(appids []int, bias bool) (int, error) {
	pool, weights, err := s.pool(wheel.Filters{}, bias)
	if err != nil {
		return -1, err
	}
	byID := map[int]library.Game{}
	for _, g := range pool {
		byID[g.AppID] = g
	}
	entries := make([]library.Game, 0, len(appids))
	for _, id := range appids {
		if g, ok := byID[id]; ok {
			entries = append(entries, g)
		}
	}
	if len(entries) != len(appids) {
		return -1, nil // the wheel is stale; the frontend redraws on library changes
	}
	return wheel.Pick(entries, weights, newRand()), nil
}

// pool is the filtered eligible pool and, with the taste-bias switch, its weights.
func (s *WheelService) pool(f wheel.Filters, bias bool) ([]library.Game, wheel.Weights, error) {
	games, settings, err := s.C.gamesAndSettings()
	if err != nil {
		return nil, nil, err
	}
	pool := wheel.Filter(recommend.Pool(games, limits(settings)), f)
	if !bias {
		return pool, nil, nil
	}
	p := recommend.BuildProfile(games, settings.Stoplist, settings.Pinned, settings.Avoided, nil)
	weights := wheel.Weights{}
	// Taste bias ignores the review minimum so it can still weight lightly reviewed games.
	for _, r := range (recommend.TagEngine{}).Recommend(p, pool) {
		weights[r.Game.AppID] = r.Score
	}
	return pool, weights, nil
}

func newRand() *rand.Rand { return rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())) }

// ---- Stats (T1) ----

// StatsService serves the Stats screen.
type StatsService struct{ C *Core }

func (s *StatsService) Stats() (stats.Stats, error) {
	games, settings, err := s.C.gamesAndSettings()
	if err != nil {
		return stats.Stats{}, err
	}
	return stats.Compute(s.C.DB, games, settings, s.C.now())
}

// ---- Settings (G1) ----

// SettingsService reads and saves preferences. The API key is handled by SetupService.
type SettingsService struct{ C *Core }

func (s *SettingsService) Get() (store.Settings, error) { return s.C.Settings() }

// Save stores preferences, keeping the account and sync fields as they are.
func (s *SettingsService) Save(in store.Settings) error {
	cur, err := s.C.Settings()
	if err != nil {
		return err
	}
	in.SteamID, in.ProfileURL, in.FirstSyncAt, in.LastSyncAt = cur.SteamID, cur.ProfileURL, cur.FirstSyncAt, cur.LastSyncAt
	tx, err := s.C.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := store.WriteSettings(tx, in); err != nil {
		return err
	}
	if !slices.Equal(in.HiddenPatterns, cur.HiddenPatterns) {
		if err := library.RecomputeHidden(tx, in.HiddenPatterns); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.C.Emit(EventLibraryChanged, nil)
	return nil
}

// limits are the R3 hours limits from settings.
func limits(s store.Settings) recommend.Limits {
	return recommend.Limits{Shelved: s.ShelvedHours, Endless: s.EndlessHours}
}
