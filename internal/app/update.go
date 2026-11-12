package app

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"somanygames/internal/update"
)

// EventUpdateProgress carries update.Progress while an update downloads.
const EventUpdateProgress = "update:progress"

// UpdateService checks for and installs a newer release, only when the owner asks (ADR 0021).
type UpdateService struct {
	C *Core

	mu      sync.Mutex
	checked *update.Release // the newer release the last check found
	busy    bool
}

func (s *UpdateService) updater() *update.Updater {
	ua := "SoManyGames/" + s.C.Version
	httpc := &http.Client{}
	return &update.Updater{
		Current:   s.C.Version,
		Source:    &update.GitHub{HTTP: httpc, UserAgent: ua, BaseURL: "https://api.github.com", Repo: update.Repo},
		HTTP:      httpc,
		UserAgent: ua,
		Dir:       s.C.UpdateDir,
		PublicKey: update.DecodedKey(),
		Installed: update.RunningInstalled(),
		Start:     update.StartDetached,
	}
}

// Check asks GitHub for the latest release.
func (s *UpdateService) Check() (update.Status, error) {
	ctx, cancel := context.WithTimeout(s.C.Context(), 20*time.Second)
	defer cancel()
	st, rel, err := s.updater().Check(ctx)
	if err != nil {
		return update.Status{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checked = nil
	if st.Available {
		s.checked = &rel
	}
	return st, nil
}

// Install downloads and verifies the release found by the last Check, starts its installer
// and quits; the installer reopens the app when it's done.
func (s *UpdateService) Install() error {
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return errors.New("an update is already downloading")
	}
	if s.checked == nil {
		s.mu.Unlock()
		return errors.New("check for updates first")
	}
	rel := *s.checked
	s.busy = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.busy = false; s.mu.Unlock() }()

	u := s.updater()
	if !u.Installed {
		return errors.New("this copy isn't installed; download the update from the release page")
	}
	ctx, cancel := context.WithTimeout(s.C.Context(), 15*time.Minute)
	defer cancel()
	path, err := u.Download(ctx, rel, func(p update.Progress) { s.C.Emit(EventUpdateProgress, p) })
	if err != nil {
		return err
	}
	if err := u.Install(path); err != nil {
		return err
	}
	go s.C.Quit()
	return nil
}

// OpenReleasePage opens the latest release on GitHub, for the portable exe.
func (s *UpdateService) OpenReleasePage() error { return s.C.OpenURL(update.ReleasesPage) }
